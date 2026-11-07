package udf

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/viant/toolbox/data"
	"github.com/xuri/excelize/v2"
)

const (
	xlsxZipLimit        = 64 << 20
	xlsxTextLimit       = 16 << 20
	xlsxCellLimit       = 100_000
	xlsxSpreadsheetNS   = "http://schemas.openxmlformats.org/spreadsheetml/2006/main"
	xlsxRelationshipsNS = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
)

// NewXlsxReader builds a reader for a static grid delimited by a worksheet AutoFilter.
func NewXlsxReader(args ...interface{}) (func(interface{}, data.Map) (interface{}, error), error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("XlsxReader requires one options map")
	}
	options, ok := args[0].(map[string]interface{})
	sheet, _ := options["sheet"].(string)
	if !ok || len(options) != 2 || strings.TrimSpace(sheet) == "" || options["range"] != "autoFilter" {
		return nil, fmt.Errorf("XlsxReader requires a worksheet name and range: autoFilter")
	}
	return func(source interface{}, _ data.Map) (interface{}, error) {
		content, ok := source.([]byte)
		if !ok || len(content) > xlsxZipLimit {
			return nil, fmt.Errorf("XlsxReader requires XLSX bytes (at most %d)", xlsxZipLimit)
		}
		return readXlsxGrid(content, sheet)
	}, nil
}

type xlsxRelations struct {
	Items []struct {
		ID     string `xml:"Id,attr"`
		Type   string `xml:"Type,attr"`
		Target string `xml:"Target,attr"`
		Mode   string `xml:"TargetMode,attr"`
	} `xml:"Relationship"`
}

func xlsxPartBytes(book *excelize.File, name string) ([]byte, error) {
	content, ok := book.Pkg.Load(name)
	if !ok {
		return nil, fmt.Errorf("missing XLSX package part %q", name)
	}
	return content.([]byte), nil
}

func xlsxRelatedPart(book *excelize.File, owner, kind, id string) (string, error) {
	relsPath := "_rels/.rels"
	base := ""
	if owner != "" {
		base = path.Dir(owner)
		relsPath = path.Join(base, "_rels", path.Base(owner)+".rels")
	}
	// Use the relationship records Excelize already decoded and used to
	// bind its worksheet names. Its concrete relationship type is private.
	parsed, ok := book.Relationships.Load(relsPath)
	if !ok {
		return "", fmt.Errorf("missing XLSX relationships %q", relsPath)
	}
	data, err := xml.Marshal(parsed)
	if err != nil {
		return "", err
	}
	var rels xlsxRelations
	if err := xml.Unmarshal(data, &rels); err != nil {
		return "", err
	}
	var part string
	relType := xlsxRelationshipsNS + "/" + kind
	for _, item := range rels.Items {
		if (id != "" && item.ID != id) || (id == "" && item.Type != relType) {
			continue
		}
		if part != "" || item.Type != relType || (item.Mode != "" && item.Mode != "Internal") {
			return "", fmt.Errorf("invalid or ambiguous XLSX %s relationship", kind)
		}
		// Resolve only paths inside the package, never external URLs.
		target, err := url.Parse(item.Target)
		if err != nil || target.Path == "" || target.Path != item.Target || target.IsAbs() || target.Opaque != "" || target.Host != "" ||
			target.RawQuery != "" || target.ForceQuery || target.Fragment != "" ||
			strings.ContainsAny(target.Path, "\\\x00") {
			return "", fmt.Errorf("invalid XLSX %s target", kind)
		}
		if strings.HasPrefix(target.Path, "/") {
			part = path.Clean(strings.TrimPrefix(target.Path, "/"))
		} else {
			part = path.Join(base, target.Path)
		}
		if part == "." || part == ".." || strings.HasPrefix(part, "../") {
			return "", fmt.Errorf("XLSX %s target is outside the package", kind)
		}
		if owner == "" && part != strings.TrimPrefix(item.Target, "/") {
			return "", fmt.Errorf("invalid XLSX %s target", kind)
		}
	}
	if part == "" {
		return "", fmt.Errorf("missing XLSX %s relationship", kind)
	}
	return part, nil
}

func xlsxSheetPart(book *excelize.File, sheet string) (string, error) {
	owner, err := xlsxRelatedPart(book, "", "officeDocument", "")
	if err != nil {
		return "", err
	}
	var id string
	for _, candidate := range book.WorkBook.Sheets.Sheet {
		// Excelize looks up sheets case-insensitively, so exclude case-only
		// collisions while requiring the requested name to match exactly.
		if strings.EqualFold(candidate.Name, sheet) {
			if candidate.Name != sheet || id != "" || candidate.ID == "" {
				return "", fmt.Errorf("invalid or ambiguous XLSX worksheet %q", sheet)
			}
			id = candidate.ID
		}
	}
	if id == "" {
		return "", fmt.Errorf("missing XLSX worksheet %q", sheet)
	}
	return xlsxRelatedPart(book, owner, "worksheet", id)
}

type xlsxRange struct{ left, top, right, bottom int }

var xlsxA1Ref = regexp.MustCompile(`^\$?[A-Z]{1,3}\$?[1-9][0-9]{0,6}$`)

func parseXlsxRange(ref string) (xlsxRange, error) {
	start, end, ok := strings.Cut(ref, ":")
	if !ok {
		end = start
	}
	if !xlsxA1Ref.MatchString(start) || !xlsxA1Ref.MatchString(end) {
		return xlsxRange{}, fmt.Errorf("invalid XLSX range %q", ref)
	}
	left, top, err := excelize.CellNameToCoordinates(start)
	if err != nil {
		return xlsxRange{}, fmt.Errorf("invalid XLSX range %q", ref)
	}
	right, bottom, err := excelize.CellNameToCoordinates(end)
	if err != nil || left > right || top > bottom || right > excelize.MaxColumns || bottom > excelize.TotalRows {
		return xlsxRange{}, fmt.Errorf("invalid XLSX range %q", ref)
	}
	return xlsxRange{left, top, right, bottom}, nil
}

type xlsxCellMetadata struct {
	Ref     string    `xml:"r,attr"`
	Type    string    `xml:"t,attr"`
	Formula *struct{} `xml:"f"`
	Value   *string   `xml:"v"` // Only used to validate shared-string indices before Excelize indexes its table.
	col     int
	row     int
}

func checkXlsxZip(content []byte) error {
	archive, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return err
	}
	// Excelize 2.8 sums sizes as signed integers. Check the central
	// directory's original unsigned sizes before passing it the ZIP.
	remaining := uint64(xlsxZipLimit)
	seen := make(map[string]bool)
	for _, entry := range archive.File {
		if entry.UncompressedSize64 > remaining {
			return fmt.Errorf("unpacked XLSX exceeds %d bytes", xlsxZipLimit)
		}
		remaining -= entry.UncompressedSize64
		if seen[entry.Name] || strings.Contains(entry.Name, "\\") {
			return fmt.Errorf("ambiguous XLSX package entry %q", entry.Name)
		}
		seen[entry.Name] = true
	}
	return nil
}

// scanXlsxElements visits direct children of a consumed start tag. Each
// visitor consumes its element, so unknown subtrees never become metadata.
func scanXlsxElements(d *xml.Decoder, visit func(xml.StartElement) error) error {
	for {
		token, err := d.Token()
		if err != nil {
			return err
		}
		switch element := token.(type) {
		case xml.StartElement:
			if err := visit(element); err != nil {
				return err
			}
		case xml.EndElement:
			return nil
		}
	}
}

func scanXlsxWorksheet(source []byte, visit func(*xml.Decoder, xml.StartElement) error) error {
	d := xml.NewDecoder(bytes.NewReader(source))
	root := false
	for {
		token, err := d.Token()
		if err == io.EOF && root {
			return nil
		}
		if err != nil {
			return fmt.Errorf("invalid XLSX worksheet XML: %w", err)
		}
		switch element := token.(type) {
		case xml.StartElement:
			if root || element.Name != (xml.Name{Space: xlsxSpreadsheetNS, Local: "worksheet"}) {
				return fmt.Errorf("unsupported XLSX worksheet root")
			}
			root = true
			if err := scanXlsxElements(d, func(child xml.StartElement) error { return visit(d, child) }); err != nil {
				return err
			}
		case xml.CharData:
			if strings.TrimSpace(string(element)) != "" {
				return fmt.Errorf("invalid text outside XLSX worksheet root")
			}
		}
	}
}

func xlsxAttribute(element xml.StartElement, name string) (string, error) {
	var result string
	found := false
	for _, attr := range element.Attr {
		if attr.Name.Local != name {
			continue
		}
		if found || attr.Name.Space != "" {
			return "", fmt.Errorf("ambiguous XLSX %s attribute %q", element.Name.Local, name)
		}
		result, found = attr.Value, true
	}
	return result, nil
}

func xlsxAutoFilter(worksheet []byte) (string, xlsxRange, error) {
	var ref string
	err := scanXlsxWorksheet(worksheet, func(d *xml.Decoder, element xml.StartElement) error {
		if element.Name == (xml.Name{Space: xlsxSpreadsheetNS, Local: "autoFilter"}) {
			if ref != "" {
				return fmt.Errorf("ambiguous XLSX worksheet AutoFilter range")
			}
			var err error
			if ref, err = xlsxAttribute(element, "ref"); err != nil {
				return err
			}
			if ref == "" {
				return fmt.Errorf("missing XLSX worksheet AutoFilter range")
			}
		}
		return d.Skip()
	})
	if err != nil {
		return "", xlsxRange{}, err
	}
	if ref == "" {
		return "", xlsxRange{}, fmt.Errorf("missing XLSX worksheet AutoFilter range")
	}
	extent, err := parseXlsxRange(ref)
	return ref, extent, err
}

// Excelize's cell getters expand sparse worksheets. Bound the highest row
// and each row's highest column before any getter can allocate those slots.
// Only retain metadata for the selected cells; Excelize decodes their values.
func selectedXlsxCells(worksheet []byte, extent xlsxRange) ([]xlsxCellMetadata, error) {
	var cells []xlsxCellMetadata
	previousRow, columns := 0, 0
	sheetData := false
	err := scanXlsxWorksheet(worksheet, func(d *xml.Decoder, element xml.StartElement) error {
		switch element.Name.Local {
		case "sheetData":
			if sheetData {
				return fmt.Errorf("ambiguous XLSX worksheet sheetData")
			}
			sheetData = true
			return scanXlsxElements(d, func(row xml.StartElement) error {
				if row.Name.Local != "row" {
					return d.Skip()
				}
				ref, err := xlsxAttribute(row, "r")
				if err != nil {
					return err
				}
				rowIndex, err := strconv.Atoi(ref)
				if err != nil || rowIndex <= previousRow || ref != strconv.Itoa(rowIndex) {
					return fmt.Errorf("invalid or out-of-order XLSX row %q", ref)
				}
				previousRow = rowIndex
				if rowIndex > xlsxCellLimit-columns {
					return fmt.Errorf("expanded XLSX worksheet exceeds %d row and cell slots", xlsxCellLimit)
				}
				lastColumn := 0
				err = scanXlsxElements(d, func(element xml.StartElement) error {
					if element.Name.Local != "c" {
						return d.Skip()
					}
					ref, err := xlsxAttribute(element, "r")
					if err != nil {
						return err
					}
					column, cellRow, err := excelize.CellNameToCoordinates(ref)
					if err != nil || column <= lastColumn || cellRow != rowIndex {
						return fmt.Errorf("invalid or out-of-order XLSX worksheet cell %q", ref)
					}
					canonical, err := excelize.CoordinatesToCellName(column, cellRow)
					if err != nil || canonical != ref {
						return fmt.Errorf("invalid XLSX worksheet cell reference %q", ref)
					}
					lastColumn = column
					if column > xlsxCellLimit-rowIndex-columns {
						return fmt.Errorf("expanded XLSX worksheet exceeds %d row and cell slots", xlsxCellLimit)
					}
					if rowIndex < extent.top || rowIndex > extent.bottom || column < extent.left || column > extent.right {
						return d.Skip()
					}
					cell := xlsxCellMetadata{col: column, row: rowIndex}
					if err := d.DecodeElement(&cell, &element); err != nil {
						return err
					}
					cells = append(cells, cell)
					return nil
				})
				columns += lastColumn
				return err
			})
		case "mergeCells":
			return scanXlsxElements(d, func(merge xml.StartElement) error {
				if merge.Name.Local != "mergeCell" {
					return d.Skip()
				}
				ref, err := xlsxAttribute(merge, "ref")
				if err != nil {
					return err
				}
				m, err := parseXlsxRange(ref)
				if err != nil {
					return err
				}
				if m.left <= extent.right && m.right >= extent.left && m.top <= extent.bottom && m.bottom >= extent.top {
					return fmt.Errorf("merged XLSX cells intersect AutoFilter range: %s", ref)
				}
				return d.Skip()
			})
		default:
			return d.Skip()
		}
	})
	if err != nil {
		return nil, err
	}
	if !sheetData {
		return nil, fmt.Errorf("missing XLSX worksheet sheetData")
	}
	return cells, nil
}

func readXlsxGrid(content []byte, sheet string) (map[string]interface{}, error) {
	if err := checkXlsxZip(content); err != nil {
		return nil, fmt.Errorf("open XLSX workbook: %w", err)
	}
	book, err := excelize.OpenReader(bytes.NewReader(content), excelize.Options{
		RawCellValue: true, UnzipSizeLimit: xlsxZipLimit, UnzipXMLSizeLimit: xlsxZipLimit,
	})
	if book != nil {
		defer book.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("open XLSX workbook: %w", err)
	}
	part, err := xlsxSheetPart(book, sheet)
	if err != nil {
		return nil, err
	}
	worksheet, err := xlsxPartBytes(book, part)
	if err != nil {
		return nil, err
	}
	ref, extent, err := xlsxAutoFilter(worksheet)
	if err != nil {
		return nil, err
	}
	width, height := extent.right-extent.left+1, extent.bottom-extent.top+1
	if height > xlsxCellLimit/width {
		return nil, fmt.Errorf("XLSX AutoFilter range exceeds %d cells", xlsxCellLimit)
	}
	cells, err := selectedXlsxCells(worksheet, extent)
	if err != nil {
		return nil, err
	}
	rows := make([][]interface{}, height)
	types := make([][]string, height)
	for i := range rows {
		rows[i] = make([]interface{}, width)
		types[i] = make([]string, width)
		for j := range types[i] {
			types[i][j] = "blank"
		}
	}
	textRemaining := xlsxTextLimit
	for _, cell := range cells {
		value, typ, err := readXlsxCell(book, sheet, cell)
		if err != nil {
			return nil, err
		}
		if text, ok := value.(string); ok {
			if len(text) > textRemaining {
				return nil, fmt.Errorf("decoded XLSX cell text exceeds %d bytes", xlsxTextLimit)
			}
			textRemaining -= len(text)
		}
		rows[cell.row-extent.top][cell.col-extent.left] = value
		types[cell.row-extent.top][cell.col-extent.left] = typ
	}
	return map[string]interface{}{"sheet": sheet, "range": ref, "byteCount": len(content), "rows": rows, "types": types}, nil
}

func readXlsxCell(book *excelize.File, sheet string, cell xlsxCellMetadata) (interface{}, string, error) {
	if cell.Formula != nil {
		return nil, "", fmt.Errorf("formula XLSX cell %s is unsupported", cell.Ref)
	}
	if cell.Type == "e" {
		return nil, "", fmt.Errorf("error XLSX cell %s is unsupported", cell.Ref)
	}
	var index int
	if cell.Type == "s" {
		var err error
		if cell.Value == nil {
			return nil, "", fmt.Errorf("missing shared string index in XLSX cell %s", cell.Ref)
		}
		index, err = strconv.Atoi(strings.TrimSpace(*cell.Value))
		if err != nil || index < 0 {
			return nil, "", fmt.Errorf("invalid shared string index in XLSX cell %s", cell.Ref)
		}
	}
	raw, err := book.GetCellValue(sheet, cell.Ref, excelize.Options{RawCellValue: true})
	if err != nil {
		return nil, "", fmt.Errorf("read XLSX cell %s: %w", cell.Ref, err)
	}
	switch cell.Type {
	case "s":
		if book.SharedStrings == nil || index >= len(book.SharedStrings.SI) {
			return nil, "", fmt.Errorf("shared string index is out of range in XLSX cell %s", cell.Ref)
		}
		return raw, "string", nil
	case "str", "inlineStr":
		return raw, "string", nil
	case "b":
		if raw == "0" || raw == "1" {
			return raw == "1", "boolean", nil
		}
	case "d":
		if raw != "" {
			return raw, "date", nil
		}
	case "", "n":
		if raw == "" {
			return nil, "blank", nil
		}
		number, err := strconv.ParseFloat(raw, 64)
		if err == nil && !math.IsNaN(number) && !math.IsInf(number, 0) && !strings.ContainsAny(raw, "xX_") {
			return number, "number", nil
		}
	default:
		return nil, "", fmt.Errorf("unsupported XLSX cell type %q in %s", cell.Type, cell.Ref)
	}
	return nil, "", fmt.Errorf("invalid XLSX cell value in %s", cell.Ref)
}
