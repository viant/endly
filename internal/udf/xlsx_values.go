package udf

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

type xlsxAttributeType uint8

const (
	xlsxInt xlsxAttributeType = iota + 1
	xlsxUint
	xlsxUint32
	xlsxBool
	xlsxFloat
)

// Excelize 2.8 Rows.Columns discards XML errors from its private cell decoder.
// Validate attributes that the decoder parses into numbers and booleans.
// Excelize still decodes the actual cell, formula cache and rich text values.
var xlsxTypedAttrs = map[string]map[string]xlsxAttributeType{
	"c":          {"s": xlsxInt, "cm": xlsxUint, "vm": xlsxUint, "ph": xlsxBool},
	"f":          {"si": xlsxInt, "aca": xlsxBool, "dt2D": xlsxBool, "dtr": xlsxBool, "del1": xlsxBool, "del2": xlsxBool, "ca": xlsxBool, "bx": xlsxBool},
	"charset":    {"val": xlsxInt},
	"family":     {"val": xlsxInt},
	"sz":         {"val": xlsxFloat},
	"color":      {"auto": xlsxBool, "indexed": xlsxInt, "theme": xlsxInt, "tint": xlsxFloat},
	"rPh":        {"sb": xlsxUint32, "eb": xlsxUint32},
	"phoneticPr": {"fontId": xlsxInt},
}

type xlsxTypedAttributeError struct {
	location string
	cause    error
}

func (e xlsxTypedAttributeError) Error() string {
	category := strconv.ErrSyntax
	if errors.Is(e.cause, strconv.ErrRange) {
		category = strconv.ErrRange
	}
	return e.location + ": " + category.Error()
}

func (e xlsxTypedAttributeError) Unwrap() error { return e.cause }

func validateXlsxTypedAttrs(element xml.StartElement, ref string) error {
	types := xlsxTypedAttrs[element.Name.Local]
	if types == nil {
		return nil
	}
	for _, attr := range element.Attr {
		kind := types[attr.Name.Local]
		if kind == 0 {
			continue
		}
		value, err := xlsxAttribute(element, attr.Name.Local)
		if err != nil {
			return fmt.Errorf("invalid XLSX typed attribute %s@%s in %s: %w", element.Name.Local, attr.Name.Local, ref, err)
		}
		// encoding/xml treats a truly empty attribute as the zero value.
		if value == "" {
			continue
		}
		value = strings.TrimSpace(value)
		switch kind {
		case xlsxInt:
			_, err = strconv.ParseInt(value, 10, strconv.IntSize)
		case xlsxUint:
			_, err = strconv.ParseUint(value, 10, strconv.IntSize)
		case xlsxUint32:
			_, err = strconv.ParseUint(value, 10, 32)
		case xlsxBool:
			_, err = strconv.ParseBool(value)
		case xlsxFloat:
			_, err = strconv.ParseFloat(value, 64)
		}
		if err != nil {
			return xlsxTypedAttributeError{
				location: fmt.Sprintf("invalid XLSX typed attribute %s@%s in %s", element.Name.Local, attr.Name.Local, ref),
				cause:    err,
			}
		}
	}
	return nil
}

type xlsxSharedStringCost struct {
	bytes int
	runs  int
}

// Excelize v2.8 visits every rich-text run each time it materializes a shared
// string. An item's XML size also bounds the size of its decoded text.
func xlsxSharedStringCosts(book *excelize.File) ([]xlsxSharedStringCost, error) {
	value, exists := book.Pkg.Load("xl/sharedStrings.xml")
	if !exists {
		return nil, nil
	}
	raw := value.([]byte)
	var costs []xlsxSharedStringCost
	d := xml.NewDecoder(bytes.NewReader(raw))
	err := scanXlsxPart(d, xml.Name{Space: xlsxSpreadsheetNS, Local: "sst"}, func(d *xml.Decoder, root xml.StartElement) error {
		return scanXlsxElements(d, func(item xml.StartElement) error {
			if item.Name.Local == "si" && item.Name.Space != xlsxSpreadsheetNS {
				return fmt.Errorf("ambiguous XLSX shared string item namespace")
			}
			if item.Name.Local != "si" {
				return d.Skip()
			}
			start := d.InputOffset()
			cost := xlsxSharedStringCost{}
			if err := scanXlsxElements(d, func(child xml.StartElement) error {
				// The native decoder matches rich-text runs by local name.
				if child.Name.Local == "r" {
					cost.runs++
				}
				return d.Skip()
			}); err != nil {
				return err
			}
			cost.bytes = int(d.InputOffset() - start)
			costs = append(costs, cost)
			return nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("inspect XLSX shared strings: %w", err)
	}
	return costs, nil
}

func (b *xlsxBudget) consumeSharedString(book *excelize.File, cell xlsxCellMetadata) (int, int, error) {
	if cell.Value == nil {
		return 0, 0, fmt.Errorf("missing shared string index in XLSX cell %s", cell.Ref)
	}
	index, err := strconv.Atoi(strings.TrimSpace(*cell.Value))
	if err != nil || index < 0 {
		return 0, 0, fmt.Errorf("invalid shared string index in XLSX cell %s", cell.Ref)
	}
	if !b.sharedReady {
		b.sharedCosts, err = xlsxSharedStringCosts(book)
		if err != nil {
			return 0, 0, err
		}
		b.sharedReady = true
	}
	if index >= len(b.sharedCosts) {
		return 0, 0, fmt.Errorf("shared string index is out of range in XLSX cell %s", cell.Ref)
	}
	cost := b.sharedCosts[index]
	if cost.runs > xlsxSharedRunLimit-b.sharedRuns {
		return 0, 0, fmt.Errorf("XLSX shared string materialization work exceeds %d rich-text runs", xlsxSharedRunLimit)
	}
	b.sharedRuns += cost.runs
	return index, cost.bytes, nil
}

// readValues uses Excelize's streaming row reader for batch mode. The preflight
// scan has already validated the sheet XML and isolated cells carrying values,
// so style-only padding is never stored or expanded into a worksheet grid.
func (s *xlsxSheetContents) readValues(book *excelize.File, sheet string, budget *xlsxBudget) (result map[xlsxPosition]xlsxObservedCell, err error) {
	result = make(map[xlsxPosition]xlsxObservedCell, len(s.cells))
	if len(s.cells) == 0 {
		return result, nil
	}
	type observedRow struct {
		positions   []xlsxPosition
		sharedBytes int
		lastColumn  int
	}
	byRow := make(map[int]*observedRow)
	lastRow := 0
	for pos, cell := range s.cells {
		if cell.problem != nil {
			return nil, cell.problem
		}
		lastRow = max(lastRow, pos.row)
		row := byRow[pos.row]
		if row == nil {
			row = &observedRow{}
			byRow[pos.row] = row
		}
		row.positions = append(row.positions, pos)
		row.lastColumn = max(row.lastColumn, pos.col)
		if cell.Type == "s" {
			_, size, err := budget.consumeSharedString(book, cell)
			if err != nil {
				return nil, err
			}
			if size > xlsxTextLimit-row.sharedBytes {
				return nil, fmt.Errorf("decoded XLSX cell text exceeds %d bytes in row %d", xlsxTextLimit, pos.row)
			}
			row.sharedBytes += size
		}
	}
	// Count iterator steps and all column slots Rows.Columns may allocate,
	// including blank columns leading up to a far-right populated cell.
	work := lastRow
	for _, row := range byRow {
		work += row.lastColumn
		if work > xlsxStreamSlotLimit-budget.streamSlots {
			return nil, fmt.Errorf("expanded XLSX streaming work exceeds %d row and cell slots", xlsxStreamSlotLimit)
		}
	}
	budget.streamSlots += work

	rows, err := book.Rows(sheet)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			result = nil
			err = errors.Join(err, closeErr)
		}
	}()
	readText := &xlsxBudget{}
	for rowNum := 1; rowNum <= lastRow; rowNum++ {
		if !rows.Next() {
			if err := rows.Error(); err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("XLSX values ended before observed row %d", lastRow)
		}
		row := byRow[rowNum]
		if row == nil {
			continue
		}
		columns, err := rows.Columns(excelize.Options{RawCellValue: true})
		if err != nil {
			return nil, err
		}
		for _, pos := range row.positions {
			var raw string
			if pos.col <= len(columns) {
				raw = columns[pos.col-1]
			}
			if err := readText.consumeText(raw); err != nil {
				return nil, err
			}
			cell := s.cells[pos]
			// For scalar values the raw native result must match the validated
			// XML value. Detect a dropped or shifted cell instead of emitting
			// a valid-looking blank.
			if cell.Type != "s" && cell.Type != "inlineStr" && cell.Value != nil && raw != *cell.Value {
				return nil, fmt.Errorf("native XLSX value disagrees with XML in cell %s", cell.Ref)
			}
			result[pos], err = observeXlsxCell(cell, raw)
			if err != nil {
				return nil, err
			}
		}
	}
	return result, rows.Error()
}

func observeXlsxCell(cell xlsxCellMetadata, raw string) (xlsxObservedCell, error) {
	if cell.Formula != nil {
		cached := xlsxObservedCell{kind: "formula"}
		if cell.Value == nil {
			return cached, nil
		}
		if *cell.Value == "" {
			cached.value = ""
			return cached, nil
		}
		if cell.Type == "e" {
			cached.value = *cell.Value
			return cached, nil
		}
		cell.Formula = nil
		value, _, err := typedXlsxCell(cell, raw)
		cached.value = value
		return cached, err
	}
	if cell.Type == "e" {
		return xlsxObservedCell{kind: "error"}, nil
	}
	value, kind, err := typedXlsxCell(cell, raw)
	return xlsxObservedCell{value, kind}, err
}
