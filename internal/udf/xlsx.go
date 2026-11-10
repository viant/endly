package udf

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
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

// NewXlsxReader builds a reader for static grids and optional workbook metadata.
func NewXlsxReader(args ...interface{}) (func(interface{}, data.Map) (interface{}, error), error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("XlsxReader requires one options map")
	}
	options, ok := args[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("XlsxReader requires one options map")
	}
	request, err := newXlsxRequest(options)
	if err != nil {
		return nil, err
	}
	return func(source interface{}, _ data.Map) (interface{}, error) {
		content, ok := source.([]byte)
		if !ok || len(content) > xlsxZipLimit {
			return nil, fmt.Errorf("XlsxReader requires XLSX bytes (at most %d)", xlsxZipLimit)
		}
		return readXlsxWorkbook(content, request)
	}, nil
}

type xlsxRelation struct {
	ID     string `xml:"Id,attr"`
	Type   string `xml:"Type,attr"`
	Target string `xml:"Target,attr"`
	Mode   string `xml:"TargetMode,attr"`
}

type xlsxRelations struct {
	Items []xlsxRelation `xml:"Relationship"`
}

func xlsxPartBytes(book *excelize.File, name string) ([]byte, error) {
	content, ok := book.Pkg.Load(name)
	if !ok {
		return nil, fmt.Errorf("missing XLSX package part %q", name)
	}
	return content.([]byte), nil
}

func loadXlsxRelations(book *excelize.File, owner string) (xlsxRelations, error) {
	relsPath := "_rels/.rels"
	if owner != "" {
		relsPath = path.Join(path.Dir(owner), "_rels", path.Base(owner)+".rels")
	}
	// Use the relationships already decoded by Excelize when available.
	// Excelize does not load drawing relationships when opening the workbook.
	var raw []byte
	var err error
	if parsed, ok := book.Relationships.Load(relsPath); ok {
		raw, err = xml.Marshal(parsed)
	} else {
		raw, err = xlsxPartBytes(book, relsPath)
	}
	if err != nil {
		return xlsxRelations{}, err
	}
	var rels xlsxRelations
	if err := xml.Unmarshal(raw, &rels); err != nil {
		return xlsxRelations{}, fmt.Errorf("decode XLSX relationships %q: %w", relsPath, err)
	}
	return rels, nil
}

func (rels xlsxRelations) index(relevant map[string]bool) (map[string]xlsxRelation, error) {
	index := make(map[string]xlsxRelation, len(rels.Items))
	for _, relation := range rels.Items {
		if relevant != nil && !relevant[relation.ID] {
			continue
		}
		if _, exists := index[relation.ID]; exists || relation.ID == "" {
			return nil, fmt.Errorf("invalid or ambiguous XLSX relationship ID %q", relation.ID)
		}
		index[relation.ID] = relation
	}
	return index, nil
}

func (item xlsxRelation) internalPart(owner, kind string) (string, error) {
	if item.Type != xlsxRelationshipsNS+"/"+kind || (item.Mode != "" && item.Mode != "Internal") {
		return "", fmt.Errorf("invalid or ambiguous XLSX %s relationship", kind)
	}
	target, err := url.Parse(item.Target)
	if err != nil || target.Path == "" || target.Path != item.Target || target.IsAbs() || target.Opaque != "" || target.Host != "" ||
		target.RawQuery != "" || target.ForceQuery || target.Fragment != "" ||
		strings.ContainsRune(target.Path, '\\') || strings.ContainsRune(target.Path, 0) {
		return "", fmt.Errorf("invalid XLSX %s target", kind)
	}
	part := path.Join(path.Dir(owner), target.Path)
	if strings.HasPrefix(target.Path, "/") {
		part = path.Clean(strings.TrimPrefix(target.Path, "/"))
	}
	if part == "." || part == ".." || strings.HasPrefix(part, "../") {
		return "", fmt.Errorf("XLSX %s target is outside the package", kind)
	}
	if owner == "" && part != strings.TrimPrefix(item.Target, "/") {
		return "", fmt.Errorf("invalid XLSX %s target", kind)
	}
	return part, nil
}

func xlsxWorkbookPart(book *excelize.File) (string, error) {
	rels, err := loadXlsxRelations(book, "")
	if err != nil {
		return "", err
	}
	var owner string
	for _, rel := range rels.Items {
		if rel.Type != xlsxRelationshipsNS+"/officeDocument" {
			continue
		}
		if owner != "" {
			return "", fmt.Errorf("ambiguous XLSX officeDocument relationship")
		}
		owner, err = rel.internalPart("", "officeDocument")
		if err != nil {
			return "", err
		}
	}
	if owner == "" {
		return "", fmt.Errorf("missing XLSX officeDocument relationship")
	}
	return owner, nil
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
	Ref, Type string
	Formula   *string
	Value     *string // Retains cache presence and guards shared-string lookups.
	problem   error   // A value-level ambiguity only matters when this cell is read.
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
	return scanXlsxPart(d, xml.Name{Space: xlsxSpreadsheetNS, Local: "worksheet"}, func(d *xml.Decoder, _ xml.StartElement) error {
		return scanXlsxElements(d, func(child xml.StartElement) error { return visit(d, child) })
	})
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

func readXlsxCell(book *excelize.File, sheet string, cell xlsxCellMetadata, budget *xlsxBudget) (interface{}, string, error) {
	if cell.problem != nil {
		return nil, "", cell.problem
	}
	if cell.Formula != nil {
		return nil, "", fmt.Errorf("formula XLSX cell %s is unsupported", cell.Ref)
	}
	if cell.Type == "e" {
		return nil, "", fmt.Errorf("error XLSX cell %s is unsupported", cell.Ref)
	}
	if (cell.Type == "" || cell.Type == "n") && (cell.Value == nil || *cell.Value == "") {
		return nil, "blank", nil
	}
	var index int
	if cell.Type == "s" {
		var err error
		index, _, err = budget.consumeSharedString(book, cell)
		if err != nil {
			return nil, "", err
		}
	}
	raw, err := book.GetCellValue(sheet, cell.Ref, excelize.Options{RawCellValue: true})
	if err != nil {
		return nil, "", fmt.Errorf("read XLSX cell %s: %w", cell.Ref, err)
	}
	if cell.Type == "s" {
		if book.SharedStrings == nil || index >= len(book.SharedStrings.SI) {
			return nil, "", fmt.Errorf("shared string index is out of range in XLSX cell %s", cell.Ref)
		}
	}
	return typedXlsxCell(cell, raw)
}

func typedXlsxCell(cell xlsxCellMetadata, raw string) (interface{}, string, error) {
	if cell.Formula != nil {
		return nil, "", fmt.Errorf("formula XLSX cell %s is unsupported", cell.Ref)
	}
	if cell.Type == "e" {
		return nil, "", fmt.Errorf("error XLSX cell %s is unsupported", cell.Ref)
	}
	switch cell.Type {
	case "s":
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
