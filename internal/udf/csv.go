package udf

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
)

func NewCsvReader(args ...interface{}) (func(source interface{}, state data.Map) (interface{}, error), error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("failed to create csv reader - expected header argument")
	}
	if options, ok := args[0].(map[string]interface{}); ok && options["output"] == "rows" {
		if len(args) != 1 {
			return nil, fmt.Errorf("csv rows output expects one options argument")
		}
		var delimiter string
		if value, present := options["delimiter"]; present {
			if delimiter, ok = value.(string); !ok {
				return nil, fmt.Errorf("csv delimiter must be a string")
			}
		}
		return newCSVRowsReader(delimiter)
	}
	delimiter := ""
	header := toolbox.AsString(args[0])
	if len(args) > 1 {
		delimiter = toolbox.AsString(args[1])
	}
	if delimiter == "" {
		delimiter = ","
	}
	if !strings.Contains(header, delimiter) {
		return nil, fmt.Errorf("invalid header or delimiter: %v %v", header, delimiter)
	}
	headerFields := strings.Split(header, delimiter)

	return func(source interface{}, state data.Map) (interface{}, error) {
		content, err := csvContent(source)
		if err != nil {
			return nil, err
		}
		skipHeader := strings.HasPrefix(content, header)

		reader := csv.NewReader(strings.NewReader(content))
		reader.Comma = rune(delimiter[0])
		var records = []string{}

		if skipHeader {
			if _, err := reader.Read(); err != nil {
				return nil, err
			}
		}

		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}
			var aMap = make(map[string]interface{})
			for i, field := range headerFields {
				if i >= len(record) {
					break
				}
				aMap[field] = record[i]
			}
			JSON, err := toolbox.AsJSONText(aMap)
			if err != nil {
				return nil, err
			}
			records = append(records, JSON)
		}
		var result = strings.Join(records, "")
		return result, nil
	}, nil

}

func newCSVRowsReader(delimiter string) (func(interface{}, data.Map) (interface{}, error), error) {
	comma := ','
	if delimiter != "" {
		value, width := utf8.DecodeRuneInString(delimiter)
		if width != len(delimiter) || value == utf8.RuneError || value == 0 || value == '"' || value == '\r' || value == '\n' {
			return nil, fmt.Errorf("invalid csv delimiter %q", delimiter)
		}
		comma = value
	}
	return func(source interface{}, _ data.Map) (interface{}, error) {
		content, err := csvContent(source)
		if err != nil {
			return nil, err
		}
		if !utf8.ValidString(content) {
			return nil, fmt.Errorf("csv content is not valid UTF-8")
		}
		content = strings.TrimPrefix(content, "\ufeff")
		reader := csv.NewReader(strings.NewReader(content))
		reader.Comma = comma
		reader.FieldsPerRecord = -1
		rows := [][]string{}
		for {
			start := reader.InputOffset()
			row, err := reader.Read()
			if err != nil && err != io.EOF {
				return nil, err
			}
			// encoding/csv skips blank lines before each record and at EOF.
			// Recover only those prefixes, never newlines within a quoted cell.
			for consumed := content[start:reader.InputOffset()]; len(consumed) > 0; {
				if consumed[0] == '\n' {
					consumed = consumed[1:]
				} else if strings.HasPrefix(consumed, "\r\n") {
					consumed = consumed[2:]
				} else {
					break
				}
				rows = append(rows, []string{})
			}
			if err == io.EOF {
				return rows, nil
			}
			rows = append(rows, row)
		}
	}, nil
}

func csvContent(source interface{}) (string, error) {
	switch value := source.(type) {
	case string:
		return value, nil
	case []byte:
		return string(value), nil
	default:
		return "", fmt.Errorf("unsupported %T", source)
	}
}
