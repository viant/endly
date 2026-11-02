package bigquery

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/viant/toolbox/data"
)

// QueryRecords decodes the scalar values in a BigQuery query-response table.
// It performs no requests and needs no credentials or client configuration.
func QueryRecords(source interface{}, _ data.Map) (interface{}, error) {
	var raw []byte
	var err error
	switch value := source.(type) {
	case string:
		raw = []byte(value)
	case []byte:
		raw = value
	default:
		raw, err = json.Marshal(value)
	}
	if err != nil {
		return nil, err
	}
	var response struct {
		Schema struct{ Fields []struct{ Name, Type string } }
		Rows   []struct{ F []struct{ V interface{} } }
	}
	if err = json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("BigQuery query records: %w", err)
	}
	records := make([]map[string]interface{}, 0, len(response.Rows))
	for _, row := range response.Rows {
		record := make(map[string]interface{}, len(response.Schema.Fields))
		for index, field := range response.Schema.Fields {
			var value interface{}
			if index < len(row.F) {
				value = row.F[index].V
			}
			if value != nil {
				switch strings.ToUpper(field.Type) {
				case "INTEGER", "INT64", "FLOAT", "FLOAT64", "NUMERIC":
					value, err = strconv.ParseFloat(fmt.Sprint(value), 64)
				case "BOOLEAN", "BOOL":
					value, err = strconv.ParseBool(fmt.Sprint(value))
				}
				if err != nil {
					return nil, fmt.Errorf("BigQuery field %q (%s): %w", field.Name, field.Type, err)
				}
			}
			record[field.Name] = value
		}
		records = append(records, record)
	}
	return records, nil
}
