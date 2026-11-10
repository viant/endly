package udf

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/xuri/excelize/v2"
)

const (
	xlsxDrawingNS        = "http://schemas.openxmlformats.org/drawingml/2006/spreadsheetDrawing"
	xlsxChartNS          = "http://schemas.openxmlformats.org/drawingml/2006/chart"
	xlsxChartContentType = "application/vnd.openxmlformats-officedocument.drawingml.chart+xml"
)

func xlsxRelationshipID(element xml.StartElement) string {
	for _, attr := range element.Attr {
		if attr.Name == (xml.Name{Space: xlsxRelationshipsNS, Local: "id"}) {
			return attr.Value
		}
	}
	return ""
}

func (s *xlsxSheetContents) scanMetadata(d *xml.Decoder, element xml.StartElement) error {
	switch element.Name.Local {
	case "drawing":
		if s.drawing != "" {
			return fmt.Errorf("ambiguous XLSX worksheet drawing")
		}
		s.drawing = xlsxRelationshipID(element)
		if s.drawing == "" {
			return fmt.Errorf("missing XLSX drawing relationship ID")
		}
	case "hyperlinks":
		return scanXlsxMetadataChildren(d, func(link xml.StartElement) error {
			if link.Name != (xml.Name{Space: xlsxSpreadsheetNS, Local: "hyperlink"}) {
				return skipXlsxRowStructure(d, link, true)
			}
			ref, err := xlsxAttribute(link, "ref")
			if err != nil {
				return err
			}
			if _, err := parseXlsxRange(ref); err != nil {
				return fmt.Errorf("invalid XLSX hyperlink reference: %w", err)
			}
			if _, exists := s.hyperlinks[ref]; exists {
				return fmt.Errorf("duplicate XLSX hyperlink reference %q", ref)
			}
			location, err := xlsxAttribute(link, "location")
			if err != nil {
				return err
			}
			id := xlsxRelationshipID(link)
			if id == "" && location == "" {
				return fmt.Errorf("missing XLSX hyperlink destination for %q", ref)
			}
			s.hyperlinks[ref] = location
			if id != "" {
				s.linkIDs[ref] = id
			}
			return skipXlsxRowStructure(d, link, true)
		})
	case "sheetViews":
		// Preserve the pane attributes as written, including split fractions.
		// Excelize's public GetPanes truncates the splits to integer positions.
		return scanXlsxMetadataChildren(d, func(view xml.StartElement) error {
			if view.Name != (xml.Name{Space: xlsxSpreadsheetNS, Local: "sheetView"}) {
				return skipXlsxRowStructure(d, view, true)
			}
			return scanXlsxMetadataChildren(d, func(pane xml.StartElement) error {
				if pane.Name != (xml.Name{Space: xlsxSpreadsheetNS, Local: "pane"}) {
					return skipXlsxRowStructure(d, pane, true)
				}
				if len(s.pane) == 0 {
					for _, attr := range pane.Attr {
						if attr.Name.Space == "" {
							s.pane[attr.Name.Local] = attr.Value
						}
					}
				}
				return skipXlsxRowStructure(d, pane, true)
			})
		})
	}
	return skipXlsxRowStructure(d, element, true)
}

// Apply the worksheet's structure guard before visiting recognized metadata
// children as well as inside unrecognized metadata subtrees.
func scanXlsxMetadataChildren(d *xml.Decoder, visit func(xml.StartElement) error) error {
	return scanXlsxElements(d, func(child xml.StartElement) error {
		if err := checkXlsxOutsideRows(child, true); err != nil {
			return err
		}
		return visit(child)
	})
}

func inspectXlsxChartSheet(raw []byte) (xlsxSheetContents, error) {
	result := xlsxSheetContents{pane: map[string]string{}, hyperlinks: map[string]string{}}
	d := xml.NewDecoder(bytes.NewReader(raw))
	err := scanXlsxPart(d, xml.Name{Space: xlsxSpreadsheetNS, Local: "chartsheet"}, func(d *xml.Decoder, _ xml.StartElement) error {
		return scanXlsxElements(d, func(child xml.StartElement) error {
			if child.Name == (xml.Name{Space: xlsxSpreadsheetNS, Local: "drawing"}) {
				return result.scanMetadata(d, child)
			}
			return d.Skip()
		})
	})
	if err == nil && result.drawing == "" {
		err = fmt.Errorf("missing XLSX chart sheet drawing")
	}
	return result, err
}

type xlsxInventory struct {
	names        []map[string]interface{}
	namesBySheet map[string]map[string]string
	sheets       map[string]interface{}
	charts       map[string]xlsxChart
	drawings     map[string][]string
}

func newXlsxInventory() xlsxInventory {
	return xlsxInventory{
		names:        []map[string]interface{}{},
		namesBySheet: map[string]map[string]string{},
		sheets:       map[string]interface{}{},
		charts:       map[string]xlsxChart{},
		drawings:     map[string][]string{},
	}
}

func (i *xlsxInventory) readNames(book *excelize.File, owner string, sheets []string, budget *xlsxBudget) error {
	if book.WorkBook.DefinedNames == nil {
		return nil
	}
	// Use the native workbook's decoded names. GetDefinedName drops attributes
	// such as hidden and the original local sheet index. Preserve the original
	// XML attributes as text, including explicit default attribute values.
	raw, err := xlsxPartBytes(book, owner)
	if err != nil {
		return err
	}
	attributes, err := xlsxDefinedNameAttributes(raw, budget)
	if err != nil {
		return fmt.Errorf("read XLSX defined name attributes: %w", err)
	}
	entries := book.WorkBook.DefinedNames.DefinedName
	if len(entries) != len(attributes) {
		return fmt.Errorf("XLSX defined name count disagrees with workbook metadata")
	}
	for index, entry := range entries {
		if err := budget.consumeText(entry.Name, entry.Data, entry.Comment); err != nil {
			return err
		}
		name := map[string]interface{}{"name": entry.Name, "value": entry.Data, "attributes": attributes[index]}
		if entry.Comment != "" {
			name["comment"] = entry.Comment
		}
		if entry.Hidden {
			name["hidden"] = true
		}
		scope := "*"
		if entry.LocalSheetID != nil {
			index := *entry.LocalSheetID
			if index < 0 || index >= len(sheets) {
				return fmt.Errorf("invalid XLSX defined name scope %d", index)
			}
			scope = sheets[index]
			name["localSheetId"] = index
		}
		i.names = append(i.names, name)
		if i.namesBySheet[scope] == nil {
			i.namesBySheet[scope] = map[string]string{}
		}
		if _, exists := i.namesBySheet[scope][entry.Name]; exists {
			return fmt.Errorf("duplicate XLSX defined name %q in scope %q", entry.Name, scope)
		}
		i.namesBySheet[scope][entry.Name] = entry.Data
		if err := budget.consumeText(scope, entry.Name, entry.Data); err != nil {
			return err
		}
	}
	return nil
}

func xlsxDefinedNameAttributes(raw []byte, budget *xlsxBudget) ([]map[string]string, error) {
	var attributes []map[string]string
	d := xml.NewDecoder(bytes.NewReader(raw))
	err := scanXlsxPart(d, xml.Name{Space: xlsxSpreadsheetNS, Local: "workbook"}, func(d *xml.Decoder, _ xml.StartElement) error {
		return scanXlsxElements(d, func(group xml.StartElement) error {
			if group.Name != (xml.Name{Space: xlsxSpreadsheetNS, Local: "definedNames"}) {
				return d.Skip()
			}
			return scanXlsxElements(d, func(name xml.StartElement) error {
				if name.Name != (xml.Name{Space: xlsxSpreadsheetNS, Local: "definedName"}) {
					return d.Skip()
				}
				attrs := make(map[string]string, len(name.Attr))
				for _, attr := range name.Attr {
					if attr.Name.Space != "" {
						continue
					}
					if _, exists := attrs[attr.Name.Local]; exists {
						return fmt.Errorf("duplicate XLSX defined name attribute %q", attr.Name.Local)
					}
					if err := budget.consumeText(attr.Name.Local, attr.Value); err != nil {
						return err
					}
					attrs[attr.Name.Local] = attr.Value
				}
				attributes = append(attributes, attrs)
				return d.Skip()
			})
		})
	})
	return attributes, err
}

func (i *xlsxInventory) readSheet(book *excelize.File, sheet, part, kind string, contents *xlsxSheetContents, values map[xlsxPosition]xlsxObservedCell, budget *xlsxBudget) error {
	if err := budget.consumeGridCells(len(contents.cells)); err != nil {
		return err
	}
	if err := budget.consumeText(sheet, sheet, contents.autoFilter); err != nil {
		return err
	}
	cells := make(map[string]interface{}, len(contents.cells))
	types := make(map[string]string, len(contents.cells))
	formulas := map[string]interface{}{}
	cellErrors := map[string]string{}
	maxRow := 0
	for pos, cell := range contents.cells {
		if err := budget.consumeText(cell.Ref); err != nil {
			return err
		}
		if cell.Type == "e" {
			if cell.Value == nil {
				return fmt.Errorf("missing XLSX error code in cell %s", cell.Ref)
			}
			if err := budget.consumeText(cell.Ref, *cell.Value); err != nil {
				return err
			}
			cellErrors[cell.Ref] = *cell.Value
			cells[cell.Ref], types[cell.Ref] = nil, "error"
		}
		if cell.Formula != nil {
			if err := budget.consumeText(cell.Ref, *cell.Formula); err != nil {
				return err
			}
			cached := values[pos].value
			if text, ok := cached.(string); ok {
				if err := budget.consumeText(text); err != nil {
					return err
				}
			}
			formulas[cell.Ref] = map[string]interface{}{"expression": *cell.Formula, "cachedValue": cached}
			cells[cell.Ref], types[cell.Ref] = nil, "formula"
		} else if cell.Type != "e" {
			value := values[pos]
			if text, ok := value.value.(string); ok {
				if err := budget.consumeText(text); err != nil {
					return err
				}
			}
			if value.kind == "blank" {
				continue
			}
			cells[cell.Ref], types[cell.Ref] = value.value, value.kind
		}
		maxRow = max(maxRow, pos.row)
	}

	var chartParts []string
	var index map[string]xlsxRelation
	if contents.drawing != "" || len(contents.linkIDs) > 0 {
		rels, err := loadXlsxRelations(book, part)
		if err != nil {
			return err
		}
		index, err = rels.index(nil)
		if err != nil {
			return err
		}
		if contents.drawing != "" {
			drawing, err := index[contents.drawing].internalPart(part, "drawing")
			if err != nil {
				return err
			}
			chartParts, err = i.readDrawing(book, drawing, budget)
			if err != nil {
				return err
			}
		}
	}
	for ref, location := range contents.hyperlinks {
		if err := budget.consumeText(ref, location); err != nil {
			return err
		}
		id, linked := contents.linkIDs[ref]
		if !linked {
			continue
		}
		rel, ok := index[id]
		if !ok || rel.Type != xlsxRelationshipsNS+"/hyperlink" || rel.Target == "" {
			return fmt.Errorf("invalid XLSX hyperlink relationship for %q", ref)
		}
		separator := ""
		if location != "" {
			separator = "#"
		}
		if err := budget.consumeText(rel.Target, separator); err != nil {
			return err
		}
		// Charge the whole target and location before joining them.
		contents.hyperlinks[ref] = rel.Target + separator + location
	}
	for key, value := range contents.pane {
		if err := budget.consumeText(key, value); err != nil {
			return err
		}
	}
	if chartParts == nil {
		chartParts = []string{}
	}
	for _, chart := range chartParts {
		if err := budget.consumeText(chart); err != nil {
			return err
		}
	}
	var filter interface{}
	if contents.autoFilter != "" {
		filter = contents.autoFilter
	}
	i.sheets[sheet] = map[string]interface{}{
		"kind":       kind,
		"autoFilter": filter, "pane": contents.pane, "hyperlinks": contents.hyperlinks,
		"charts": chartParts, "cells": cells, "types": types, "formulas": formulas, "errors": cellErrors,
		"maxRow": maxRow, "formulaCount": len(formulas), "errorCount": len(cellErrors),
	}
	return nil
}

func (i *xlsxInventory) result(sheetNames []string) map[string]interface{} {
	charts := make(map[string]interface{}, len(i.charts))
	for part, chart := range i.charts {
		charts[part] = chart.result()
	}
	return map[string]interface{}{
		"sheetNames": sheetNames, "sheetCount": len(sheetNames), "sheets": i.sheets,
		"definedNames": i.names, "definedNamesBySheet": i.namesBySheet,
		"charts": charts, "chartCount": len(charts),
	}
}

func (i *xlsxInventory) readDrawing(book *excelize.File, part string, budget *xlsxBudget) ([]string, error) {
	if charts, exists := i.drawings[part]; exists {
		return charts, nil
	}
	raw, err := xlsxPartBytes(book, part)
	if err != nil {
		return nil, err
	}
	d := xml.NewDecoder(bytes.NewReader(raw))
	var ids []string
	err = scanXlsxPart(d, xml.Name{Space: xlsxDrawingNS, Local: "wsDr"}, func(d *xml.Decoder, root xml.StartElement) error {
		return walkXlsxElements(d, func(_ *xml.Decoder, element xml.StartElement) (bool, error) {
			if element.Name == (xml.Name{Space: xlsxChartNS, Local: "chart"}) {
				id := xlsxRelationshipID(element)
				if id == "" {
					return false, fmt.Errorf("missing XLSX chart relationship ID")
				}
				ids = append(ids, id)
			}
			return false, nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("XLSX drawing %q: %w", part, err)
	}
	charts := make([]string, 0, len(ids))
	if len(ids) > 0 {
		rels, err := loadXlsxRelations(book, part)
		if err != nil {
			return nil, err
		}
		index, err := rels.index(nil)
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			target, err := index[id].internalPart(part, "chart")
			if err != nil {
				return nil, err
			}
			if err := i.readChart(book, target, budget); err != nil {
				return nil, err
			}
			charts = append(charts, target)
		}
	}
	i.drawings[part] = charts
	return charts, nil
}

func (i *xlsxInventory) readChart(book *excelize.File, part string, budget *xlsxBudget) error {
	if _, exists := i.charts[part]; exists {
		return nil
	}
	if err := budget.consumeText(part); err != nil {
		return err
	}
	raw, err := xlsxPartBytes(book, part)
	if err != nil {
		return err
	}
	chart, err := inspectXlsxChart(raw, budget)
	if err != nil {
		return fmt.Errorf("XLSX chart %q: %w", part, err)
	}
	i.charts[part] = chart
	return nil
}

func (i *xlsxInventory) readDeclaredCharts(book *excelize.File, budget *xlsxBudget) error {
	if book.ContentTypes == nil {
		return nil
	}
	for _, part := range book.ContentTypes.Overrides {
		if part.ContentType != xlsxChartContentType {
			continue
		}
		relation := xlsxRelation{Type: xlsxRelationshipsNS + "/chart", Target: part.PartName}
		target, err := relation.internalPart("", "chart")
		if err != nil {
			return err
		}
		if err := i.readChart(book, target, budget); err != nil {
			return err
		}
	}
	return nil
}

// scanXlsxPart requires one OOXML root and a fully consumed document.
func scanXlsxPart(d *xml.Decoder, expected xml.Name, scan func(*xml.Decoder, xml.StartElement) error) error {
	found := false
	for {
		token, err := d.Token()
		if err == io.EOF && found {
			return nil
		}
		if err != nil {
			return err
		}
		switch element := token.(type) {
		case xml.StartElement:
			if found || element.Name != expected {
				return fmt.Errorf("invalid XLSX %s root", expected.Local)
			}
			found = true
			if err := scan(d, element); err != nil {
				return err
			}
		case xml.CharData:
			if strings.TrimSpace(string(element)) != "" {
				return fmt.Errorf("invalid text outside XLSX %s root", expected.Local)
			}
		}
	}
}

// walkXlsxElements visits descendants until the current element ends. A
// visitor reports whether it consumed a start tag's entire subtree.
func walkXlsxElements(d *xml.Decoder, visit func(*xml.Decoder, xml.StartElement) (bool, error)) error {
	depth := 1
	for depth > 0 {
		token, err := d.Token()
		if err != nil {
			return err
		}
		switch element := token.(type) {
		case xml.StartElement:
			consumed, err := visit(d, element)
			if err != nil {
				return err
			}
			if !consumed {
				depth++
			}
		case xml.EndElement:
			depth--
		}
	}
	return nil
}
