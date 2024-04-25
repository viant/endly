package table

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
)

type Exporter struct {
	parser *Parser
}

// Export exports the data in the specified format ("json" or "csv")
func (e *Exporter) Export(headers []string, format string) (interface{}, error) {
	data := e.parser.ParseTable()
	switch format {
	case "json":
		return e.exportJSON(headers, data)
	case "csv":
		return e.exportCSV(headers, data)
	case "objects":
		return e.exportObjects(headers, data), nil
	case "tabular":
		return data, nil
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// exportJSON converts the table data into JSON format
func (e *Exporter) exportJSON(headers []string, data [][]string) (string, error) {
	if len(data) < 1 {
		return "[]", nil // return empty JSON array if no data
	}
	objects := e.exportObjects(headers, data)
	jsonData, err := json.MarshalIndent(objects, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

func (e *Exporter) exportObjects(headers []string, data [][]string) []map[string]interface{} {
	offset := 0
	if len(headers) == 0 {
		headers = data[0]
		offset++
	}
	var objects []map[string]interface{}
	for _, row := range data[offset:] {
		if len(row) != len(headers) {
			continue // skip rows that do not match header length
		}
		obj := make(map[string]interface{})
		for i, header := range headers {
			if header == "" {
				continue
			}
			obj[header] = row[i]

		}
		objects = append(objects, obj)
	}
	return objects
}

// exportCSV converts the table data into CSV format
func (e *Exporter) exportCSV(headers []string, data [][]string) (string, error) {
	buffer := new(bytes.Buffer)
	writer := csv.NewWriter(buffer)
	if err := writer.Write(headers); err != nil {
		return "", err
	}
	for _, record := range data {
		if err := writer.Write(record); err != nil {
			return "", err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

// NewExporter creates a new exporter instance
func NewExporter(htmlContent string) (*Exporter, error) {
	parser, err := NewParser(htmlContent)
	if err != nil {
		return nil, err
	}
	return &Exporter{parser: parser}, nil
}
