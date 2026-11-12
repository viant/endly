package udf

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/xuri/excelize/v2"
)

type xlsxGridRequest struct {
	id, sheet, ref string
	extent         xlsxRange
}

type xlsxRequest struct {
	single, metadata   bool
	grids              []xlsxGridRequest
	metadataTextSheets map[string]bool
}

func newXlsxRequest(options map[string]interface{}) (xlsxRequest, error) {
	var result xlsxRequest
	for key := range options {
		switch key {
		case "sheet", "range", "ranges", "metadata", "metadataTextSheets":
		default:
			return result, fmt.Errorf("unknown XlsxReader option %q", key)
		}
	}
	if value, ok := options["metadata"]; ok {
		result.metadata, ok = value.(bool)
		if !ok {
			return result, fmt.Errorf("XlsxReader metadata must be a boolean")
		}
	}
	if value, exists := options["metadataTextSheets"]; exists {
		items, ok := value.([]interface{})
		if !result.metadata || !ok || len(items) == 0 || len(items) > xlsxCellLimit {
			return result, fmt.Errorf("XlsxReader metadataTextSheets requires metadata: true and between 1 and %d worksheet names", xlsxCellLimit)
		}
		result.metadataTextSheets = make(map[string]bool, len(items))
		folded := make(map[string]bool, len(items))
		for _, item := range items {
			name, ok := item.(string)
			if !ok || strings.TrimSpace(name) == "" {
				return result, fmt.Errorf("XlsxReader metadataTextSheets requires nonempty worksheet names")
			}
			key := foldXlsxSheetName(name)
			if folded[key] {
				return result, fmt.Errorf("duplicate XlsxReader metadataTextSheets worksheet %q", name)
			}
			folded[key], result.metadataTextSheets[name] = true, true
		}
	}
	rawRanges, hasRanges := options["ranges"]
	_, hasSheet := options["sheet"]
	_, hasRange := options["range"]
	if hasSheet || hasRange {
		if hasRanges || len(options) != 2 {
			return result, fmt.Errorf("XlsxReader sheet and range cannot be combined with ranges or metadata")
		}
		grid, err := newXlsxGridRequest(options, false)
		result.single, result.grids = true, []xlsxGridRequest{grid}
		return result, err
	}
	if !hasRanges {
		if result.metadata {
			return result, nil
		}
		return result, fmt.Errorf("XlsxReader requires sheet and range, ranges, or metadata: true")
	}
	items, ok := rawRanges.([]interface{})
	if !ok || len(items) == 0 || len(items) > xlsxCellLimit {
		return result, fmt.Errorf("XlsxReader requires between 1 and %d range entries", xlsxCellLimit)
	}
	ids := make(map[string]bool, len(items))
	for i, item := range items {
		opts, ok := item.(map[string]interface{})
		if !ok || len(opts) != 3 {
			return result, fmt.Errorf("XlsxReader range %d requires id, sheet and range", i+1)
		}
		grid, err := newXlsxGridRequest(opts, true)
		if err != nil {
			return result, fmt.Errorf("XlsxReader range %d: %w", i+1, err)
		}
		if ids[grid.id] {
			return result, fmt.Errorf("duplicate XlsxReader range ID %q", grid.id)
		}
		ids[grid.id] = true
		result.grids = append(result.grids, grid)
	}
	return result, nil
}

func newXlsxGridRequest(options map[string]interface{}, batch bool) (xlsxGridRequest, error) {
	var result xlsxGridRequest
	result.id, _ = options["id"].(string)
	result.sheet, _ = options["sheet"].(string)
	result.ref, _ = options["range"].(string)
	if strings.TrimSpace(result.sheet) == "" || result.ref == "" || (batch && strings.TrimSpace(result.id) == "") {
		return result, fmt.Errorf("XlsxReader requires a worksheet name and range: autoFilter or an A1 range")
	}
	if result.ref == "autoFilter" {
		return result, nil
	}
	var err error
	result.extent, err = parseXlsxRange(result.ref)
	return result, err
}

type xlsxBudget struct {
	gridCells   int
	textBytes   int
	streamSlots int
	mergeChecks int
	getterSteps int
	sharedRuns  int
	sharedCosts []xlsxSharedStringCost
	sharedReady bool
}

// Limit work separately from the number of emitted cells. Excelize's
// streaming iterator allocates dense string slices for populated rows.
const (
	xlsxStreamSlotLimit = 2_000_000
	xlsxMergeCheckLimit = 1_000_000
	xlsxGetterStepLimit = 2_000_000
	xlsxSharedRunLimit  = 2_000_000
)

func (b *xlsxBudget) consumeGetterSteps(steps int) error {
	if steps > xlsxGetterStepLimit-b.getterSteps {
		return fmt.Errorf("XLSX selected cell lookup work exceeds %d comparisons", xlsxGetterStepLimit)
	}
	b.getterSteps += steps
	return nil
}

func (b *xlsxBudget) consumeMergeChecks(merges, grids int) error {
	if merges != 0 && grids > (xlsxMergeCheckLimit-b.mergeChecks)/merges {
		return fmt.Errorf("XLSX merge intersection work exceeds %d comparisons", xlsxMergeCheckLimit)
	}
	b.mergeChecks += merges * grids
	return nil
}

func (b *xlsxBudget) consumeGridCells(count int) error {
	if count > xlsxCellLimit-b.gridCells {
		return fmt.Errorf("XLSX ranges and inventory exceed %d cells", xlsxCellLimit)
	}
	b.gridCells += count
	return nil
}

func (b *xlsxBudget) consumeText(values ...string) error {
	for _, value := range values {
		if len(value) > xlsxTextLimit-b.textBytes {
			return fmt.Errorf("decoded XLSX cell text exceeds %d bytes including metadata", xlsxTextLimit)
		}
		b.textBytes += len(value)
	}
	return nil
}

type xlsxSheetContents struct {
	cells      map[xlsxPosition]xlsxCellMetadata
	merges     []xlsxRange
	autoFilter string
	pane       map[string]string
	hyperlinks map[string]string
	linkIDs    map[string]string
	drawing    string
}

type xlsxPosition struct{ col, row int }

// Excelize's streaming reader matches row and cell tags by local name.
// Reject any such tag outside the structural positions we validate.
func checkXlsxOutsideRows(element xml.StartElement, streaming bool) error {
	switch element.Name.Local {
	case "sheetData", "row", "c":
		return fmt.Errorf("ambiguous XLSX row structure at %s", element.Name.Local)
	case "f", "v", "is":
		if streaming {
			return fmt.Errorf("ambiguous XLSX row structure at %s", element.Name.Local)
		}
	}
	return nil
}

func skipXlsxRowStructure(d *xml.Decoder, start xml.StartElement, deep bool) error {
	if err := checkXlsxOutsideRows(start, deep); err != nil {
		return err
	}
	if !deep {
		return d.Skip()
	}
	return walkXlsxElements(d, func(_ *xml.Decoder, element xml.StartElement) (bool, error) {
		return false, checkXlsxOutsideRows(element, true)
	})
}

func (s *xlsxSheetContents) scanRows(d *xml.Decoder, useGetters bool) error {
	previousRow, columns := 0, 0
	err := scanXlsxElements(d, func(row xml.StartElement) error {
		if row.Name != (xml.Name{Space: xlsxSpreadsheetNS, Local: "row"}) {
			return skipXlsxRowStructure(d, row, !useGetters)
		}
		ref, err := xlsxAttribute(row, "r")
		if err != nil {
			return err
		}
		rowIndex, err := strconv.Atoi(ref)
		if err != nil || rowIndex <= previousRow || rowIndex > excelize.TotalRows || ref != strconv.Itoa(rowIndex) {
			return fmt.Errorf("invalid or out-of-order XLSX row %q", ref)
		}
		previousRow = rowIndex
		remaining := xlsxCellLimit - columns - rowIndex
		if useGetters && remaining < 0 {
			return fmt.Errorf("expanded XLSX worksheet exceeds %d row and cell slots", xlsxCellLimit)
		}
		lastColumn := 0
		err = scanXlsxElements(d, func(element xml.StartElement) error {
			if element.Name != (xml.Name{Space: xlsxSpreadsheetNS, Local: "c"}) {
				return skipXlsxRowStructure(d, element, !useGetters)
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
			if useGetters && column > remaining {
				return fmt.Errorf("expanded XLSX worksheet exceeds %d row and cell slots", xlsxCellLimit)
			}
			cell, err := scanXlsxCell(d, element, ref, !useGetters)
			if err != nil {
				return err
			}
			if cell.problem == nil && cell.Formula == nil && (cell.Type == "" || cell.Type == "n") && (cell.Value == nil || *cell.Value == "") {
				return nil // A style-only cell does not contribute an observed value.
			}
			if len(s.cells) >= xlsxCellLimit {
				return fmt.Errorf("observed XLSX worksheet exceeds %d cells", xlsxCellLimit)
			}
			s.cells[xlsxPosition{column, rowIndex}] = cell
			return nil
		})
		columns += lastColumn
		return err
	})
	return err
}

func scanXlsxCell(d *xml.Decoder, element xml.StartElement, ref string, streaming bool) (xlsxCellMetadata, error) {
	cell := xlsxCellMetadata{Ref: ref}
	cell.Type, cell.problem = xlsxAttribute(element, "t")
	if streaming {
		if err := validateXlsxTypedAttrs(element, ref); err != nil {
			return cell, err
		}
	}
	err := scanXlsxElements(d, func(child xml.StartElement) error {
		var field **string
		switch child.Name.Local {
		case "f":
			field = &cell.Formula
		case "v":
			field = &cell.Value
		case "is":
			if child.Name.Space != xlsxSpreadsheetNS {
				cell.problem = fmt.Errorf("ambiguous XLSX cell namespace for %s in %q", child.Name.Local, ref)
			}
			if streaming {
				return walkXlsxElements(d, func(_ *xml.Decoder, element xml.StartElement) (bool, error) {
					return false, validateXlsxTypedAttrs(element, ref)
				})
			}
			return d.Skip()
		default:
			return d.Skip()
		}
		if child.Name.Space != xlsxSpreadsheetNS {
			cell.problem = fmt.Errorf("ambiguous XLSX cell namespace for %s in %q", child.Name.Local, ref)
			return d.Skip()
		}
		if *field != nil {
			cell.problem = fmt.Errorf("duplicate XLSX cell %s element in %q", child.Name.Local, ref)
			return d.Skip()
		}
		if streaming && child.Name.Local == "f" {
			if err := validateXlsxTypedAttrs(child, ref); err != nil {
				return err
			}
		}
		var value string
		if err := d.DecodeElement(&value, &child); err != nil {
			return err
		}
		*field = &value
		return nil
	})
	return cell, err
}

func inspectXlsxSheet(raw []byte, metadata, useGetters bool) (xlsxSheetContents, error) {
	result := xlsxSheetContents{
		cells: map[xlsxPosition]xlsxCellMetadata{}, pane: map[string]string{},
		hyperlinks: map[string]string{}, linkIDs: map[string]string{},
	}
	sheetData := false
	err := scanXlsxWorksheet(raw, func(d *xml.Decoder, element xml.StartElement) error {
		if element.Name.Space != xlsxSpreadsheetNS {
			switch element.Name.Local {
			case "sheetData", "mergeCells", "autoFilter":
				return fmt.Errorf("ambiguous XLSX worksheet namespace for %s", element.Name.Local)
			}
			if !sheetData && !useGetters {
				return skipXlsxRowStructure(d, element, true)
			}
			return d.Skip()
		}
		switch element.Name.Local {
		case "autoFilter":
			if result.autoFilter != "" {
				return fmt.Errorf("ambiguous XLSX worksheet AutoFilter range")
			}
			ref, err := xlsxAttribute(element, "ref")
			if err != nil {
				return err
			}
			if ref == "" {
				return fmt.Errorf("missing XLSX worksheet AutoFilter range")
			}
			if _, err := parseXlsxRange(ref); err != nil {
				return err
			}
			result.autoFilter = ref
			return skipXlsxRowStructure(d, element, !useGetters)
		case "sheetData":
			if sheetData {
				return fmt.Errorf("ambiguous XLSX worksheet sheetData")
			}
			sheetData = true
			return result.scanRows(d, useGetters)
		case "mergeCells":
			return scanXlsxElements(d, func(merge xml.StartElement) error {
				if merge.Name != (xml.Name{Space: xlsxSpreadsheetNS, Local: "mergeCell"}) {
					if merge.Name.Local == "mergeCell" {
						return fmt.Errorf("ambiguous XLSX merged cell namespace")
					}
					return skipXlsxRowStructure(d, merge, !useGetters)
				}
				ref, err := xlsxAttribute(merge, "ref")
				if err != nil {
					return err
				}
				extent, err := parseXlsxRange(ref)
				if err != nil {
					return err
				}
				if len(result.merges) >= xlsxCellLimit {
					return fmt.Errorf("too many XLSX merged ranges")
				}
				result.merges = append(result.merges, extent)
				return skipXlsxRowStructure(d, merge, !useGetters)
			})
		}
		if metadata && (element.Name.Local == "sheetViews" || element.Name.Local == "hyperlinks" || element.Name.Local == "drawing") {
			return result.scanMetadata(d, element)
		}
		if !sheetData && !useGetters {
			return skipXlsxRowStructure(d, element, true)
		}
		return d.Skip()
	})
	if err != nil {
		return result, err
	}
	if !sheetData {
		return result, fmt.Errorf("missing XLSX worksheet sheetData")
	}
	return result, nil
}

type xlsxObservedCell struct {
	value interface{}
	kind  string
}

func (s *xlsxSheetContents) grid(book *excelize.File, request xlsxGridRequest, budget *xlsxBudget, values map[xlsxPosition]xlsxObservedCell, sheetCount int) (map[string]interface{}, error) {
	ref, extent := request.ref, request.extent
	if ref == "autoFilter" {
		if s.autoFilter == "" {
			return nil, fmt.Errorf("missing XLSX worksheet AutoFilter range")
		}
		ref = s.autoFilter
		var err error
		if extent, err = parseXlsxRange(ref); err != nil {
			return nil, err
		}
	}
	width, height := extent.right-extent.left+1, extent.bottom-extent.top+1
	if height > xlsxCellLimit/width {
		return nil, fmt.Errorf("XLSX range exceeds %d cells", xlsxCellLimit)
	}
	if err := budget.consumeGridCells(width * height); err != nil {
		return nil, err
	}
	if request.id != "" {
		if err := budget.consumeText(request.id, request.sheet, ref); err != nil {
			return nil, err
		}
	}
	for _, merge := range s.merges {
		if merge.left <= extent.right && merge.right >= extent.left && merge.top <= extent.bottom && merge.bottom >= extent.top {
			return nil, fmt.Errorf("merged XLSX cells intersect range %s", ref)
		}
	}
	rows := make([][]interface{}, height)
	types := make([][]string, height)
	for i := range rows {
		rows[i], types[i] = make([]interface{}, width), make([]string, width)
		for j := range types[i] {
			types[i][j] = "blank"
			pos := xlsxPosition{extent.left + j, extent.top + i}
			cell, exists := s.cells[pos]
			if !exists {
				continue
			}
			value, err := decodeXlsxGridCell(book, request.sheet, cell, values, pos, len(s.merges), sheetCount, budget)
			if err != nil {
				return nil, err
			}
			if text, ok := value.value.(string); ok {
				if err := budget.consumeText(text); err != nil {
					return nil, err
				}
			}
			rows[i][j], types[i][j] = value.value, value.kind
		}
	}
	return map[string]interface{}{"sheet": request.sheet, "range": ref, "rows": rows, "types": types}, nil
}

func decodeXlsxGridCell(book *excelize.File, sheet string, cell xlsxCellMetadata, values map[xlsxPosition]xlsxObservedCell, pos xlsxPosition, merges, sheetCount int, budget *xlsxBudget) (xlsxObservedCell, error) {
	if value, exists := values[pos]; exists {
		if value.kind == "formula" || value.kind == "error" {
			return xlsxObservedCell{}, fmt.Errorf("%s XLSX cell %s is unsupported", value.kind, cell.Ref)
		}
		return value, nil
	}
	if values != nil {
		// Excelize also checks merges for each selected native getter call.
		// Account for those comparisons in addition to grid intersections.
		if err := budget.consumeMergeChecks(merges, 1); err != nil {
			return xlsxObservedCell{}, err
		}
		// Excelize 2.8 searches the sheet map, then scans rows and columns
		// from the start through this validated coordinate.
		if err := budget.consumeGetterSteps(sheetCount + pos.row + pos.col); err != nil {
			return xlsxObservedCell{}, err
		}
	}
	value, kind, err := readXlsxCell(book, sheet, cell, budget)
	result := xlsxObservedCell{value, kind}
	if err == nil && values != nil {
		values[pos] = result
	}
	return result, err
}

func readXlsxGrid(content []byte, sheet string) (map[string]interface{}, error) {
	return readXlsxWorkbook(content, xlsxRequest{
		single: true, grids: []xlsxGridRequest{{sheet: sheet, ref: "autoFilter"}},
	})
}

func foldXlsxSheetName(name string) string {
	return strings.Map(func(r rune) rune {
		minimum := r
		for c := unicode.SimpleFold(r); c != r; c = unicode.SimpleFold(c) {
			if c < minimum {
				minimum = c
			}
		}
		return minimum
	}, name)
}

func readXlsxWorkbook(content []byte, request xlsxRequest) (result map[string]interface{}, err error) {
	if err := checkXlsxZip(content); err != nil {
		return nil, fmt.Errorf("open XLSX workbook: %w", err)
	}
	book, err := excelize.OpenReader(bytes.NewReader(content), excelize.Options{
		RawCellValue: true, UnzipSizeLimit: xlsxZipLimit, UnzipXMLSizeLimit: xlsxZipLimit,
	})
	if book != nil {
		defer func() {
			if closeErr := book.Close(); closeErr != nil {
				result = nil
				err = errors.Join(err, fmt.Errorf("close XLSX workbook: %w", closeErr))
			}
		}()
	}
	if err != nil {
		return nil, fmt.Errorf("open XLSX workbook: %w", err)
	}
	owner, err := xlsxWorkbookPart(book)
	if err != nil {
		return nil, err
	}
	rels, err := loadXlsxRelations(book, owner)
	if err != nil {
		return nil, err
	}
	sheets := book.GetSheetList()
	bySheet := make(map[string][]xlsxGridRequest, len(request.grids))
	wantedNames := make(map[string]bool, len(request.grids))
	for _, grid := range request.grids {
		bySheet[grid.sheet] = append(bySheet[grid.sheet], grid)
		wantedNames[foldXlsxSheetName(grid.sheet)] = true
	}
	sheetIDs := make(map[string]string, len(sheets))
	foldedNames := make(map[string]string, len(sheets))
	for _, item := range book.WorkBook.Sheets.Sheet {
		folded := foldXlsxSheetName(item.Name)
		if !request.metadata && !wantedNames[folded] {
			continue
		}
		if previous, exists := foldedNames[folded]; exists {
			return nil, fmt.Errorf("invalid or ambiguous XLSX worksheet %q (also %q)", item.Name, previous)
		}
		foldedNames[folded], sheetIDs[item.Name] = item.Name, item.ID
	}
	relevant := make(map[string]bool, len(bySheet))
	for sheet := range bySheet {
		if sheetIDs[sheet] == "" {
			return nil, fmt.Errorf("missing XLSX worksheet %q", sheet)
		}
		relevant[sheetIDs[sheet]] = true
	}
	for sheet := range request.metadataTextSheets {
		if sheetIDs[sheet] == "" {
			return nil, fmt.Errorf("missing XLSX worksheet %q", sheet)
		}
	}
	if request.metadata {
		relevant = nil
	}
	byID, err := rels.index(relevant)
	if err != nil {
		return nil, err
	}
	type selectedPart struct{ sheet, part, kind string }
	var selectedParts []selectedPart
	partOwners := make(map[string]string)
	for _, sheet := range sheets {
		if !request.metadata && len(bySheet[sheet]) == 0 {
			continue
		}
		relation := byID[sheetIDs[sheet]]
		kind := "worksheet"
		if relation.Type == xlsxRelationshipsNS+"/chartsheet" {
			if len(bySheet[sheet]) != 0 || request.metadataTextSheets[sheet] {
				return nil, fmt.Errorf("XLSX sheet %q is a chart sheet and has no cell grid", sheet)
			}
			kind = "chartsheet"
		}
		part, err := relation.internalPart(owner, kind)
		if err != nil {
			return nil, fmt.Errorf("XLSX worksheet %q: %w", sheet, err)
		}
		if previous, exists := partOwners[part]; exists {
			return nil, fmt.Errorf("XLSX sheets %q and %q resolve to the same XLSX package part %q", previous, sheet, part)
		}
		partOwners[part] = sheet
		selectedParts = append(selectedParts, selectedPart{sheet, part, kind})
	}
	budget := &xlsxBudget{}
	grids := make(map[string]interface{}, len(request.grids))
	var inventory xlsxInventory
	if request.metadata {
		inventory = newXlsxInventory()
		if err := inventory.readNames(book, owner, sheets, budget); err != nil {
			return nil, err
		}
	}
	for _, selection := range selectedParts {
		sheet, part, kind := selection.sheet, selection.part, selection.kind
		worksheet, err := xlsxPartBytes(book, part)
		if err != nil {
			return nil, fmt.Errorf("XLSX worksheet %q: %w", sheet, err)
		}
		if kind == "chartsheet" {
			contents, err := inspectXlsxChartSheet(worksheet)
			if err != nil {
				return nil, fmt.Errorf("XLSX chart sheet %q: %w", sheet, err)
			}
			if err := inventory.readSheet(book, sheet, part, kind, &contents, nil, budget); err != nil {
				return nil, fmt.Errorf("XLSX chart sheet %q metadata: %w", sheet, err)
			}
			continue
		}
		contents, err := inspectXlsxSheet(worksheet, request.metadata, !request.metadata)
		if err != nil {
			return nil, fmt.Errorf("XLSX worksheet %q: %w", sheet, err)
		}
		if err := budget.consumeMergeChecks(len(contents.merges), len(bySheet[sheet])); err != nil {
			return nil, fmt.Errorf("XLSX worksheet %q: %w", sheet, err)
		}
		var values map[xlsxPosition]xlsxObservedCell
		if request.metadata {
			values, err = contents.readValues(book, sheet, budget)
			if err != nil {
				return nil, fmt.Errorf("read XLSX worksheet %q values: %w", sheet, err)
			}
		} else if !request.single {
			values = make(map[xlsxPosition]xlsxObservedCell)
		}
		for _, grid := range bySheet[sheet] {
			result, err := contents.grid(book, grid, budget, values, len(sheets))
			if err != nil {
				return nil, fmt.Errorf("XLSX worksheet %q range %q (%s): %w", sheet, grid.id, grid.ref, err)
			}
			if request.single {
				result["byteCount"] = len(content)
				return result, nil
			}
			grids[grid.id] = result
		}
		if request.metadata {
			if err := inventory.readSheet(book, sheet, part, kind, &contents, values, budget); err != nil {
				return nil, fmt.Errorf("XLSX worksheet %q metadata: %w", sheet, err)
			}
			if request.metadataTextSheets[sheet] {
				text, err := xlsxSheetText(values, budget)
				if err != nil {
					return nil, fmt.Errorf("XLSX worksheet %q metadata text: %w", sheet, err)
				}
				inventory.sheets[sheet].(map[string]interface{})["text"] = text
			}
		}
	}
	result = map[string]interface{}{"byteCount": len(content), "grids": grids}
	if request.metadata {
		if err := inventory.readDeclaredCharts(book, budget); err != nil {
			return nil, err
		}
		result["workbook"] = inventory.result(sheets)
	}
	return result, nil
}
