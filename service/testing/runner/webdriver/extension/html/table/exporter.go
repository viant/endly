package table

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
)

type Exporter struct{ parser *Parser }

// Export supports inferred table headers or explicit positional column names.
// Blank and underscore names omit a column. Irregular rows fail rather than vanish.
func (e *Exporter) Export(headers []string, format string) (interface{}, error) {
	if format == "tabular" {
		return e.parser.ParseTable(), nil
	}
	if format != "objects" && format != "json" && format != "csv" {
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
	rows := e.parser.rows()
	headers = append([]string(nil), headers...)
	if len(headers) == 0 && len(rows) > 0 {
		headers = append(headers, rows[0].cells...)
		rows = rows[1:]
	}
	for len(rows) > 0 && rows[0].header {
		rows = rows[1:]
	}
	seen := map[string]bool{}
	for i, h := range headers {
		h = strings.TrimSpace(h)
		headers[i] = h
		if h == "" || h == "_" {
			continue
		}
		if seen[h] {
			return nil, fmt.Errorf("duplicate table header %q; specify unique column names", h)
		}
		seen[h] = true
	}
	objects := make([]map[string]interface{}, 0, len(rows))
	var records [][]string
	for index, row := range rows {
		if len(row.cells) != len(headers) {
			return nil, fmt.Errorf("table row %d has %d cells; expected %d", index+1, len(row.cells), len(headers))
		}
		object := map[string]interface{}{}
		var record []string
		for i, h := range headers {
			if h == "" || h == "_" {
				continue
			}
			object[h] = row.cells[i]
			record = append(record, row.cells[i])
		}
		objects = append(objects, object)
		records = append(records, record)
	}
	switch format {
	case "objects":
		return objects, nil
	case "json":
		raw, err := json.MarshalIndent(objects, "", "  ")
		return string(raw), err
	default:
		var buf bytes.Buffer
		writer := csv.NewWriter(&buf)
		var names []string
		for _, h := range headers {
			if h != "" && h != "_" {
				names = append(names, h)
			}
		}
		if len(names) > 0 {
			if err := writer.Write(names); err != nil {
				return nil, err
			}
		}
		if err := writer.WriteAll(records); err != nil {
			return nil, err
		}
		return buf.String(), writer.Error()
	}
}
func NewExporter(content string) (*Exporter, error) {
	parser, err := NewParser(content)
	if err != nil {
		return nil, err
	}
	return &Exporter{parser: parser}, nil
}
