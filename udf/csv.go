package udf

import (
	"encoding/csv"
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"io"
	"strings"
)

func NewCsvReader(args ...interface{}) (func(source interface{}, state data.Map) (interface{}, error), error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("failed to create csv reader - expected header argument")
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
		content := ""
		switch val := source.(type) {
		case string:
			content = val
		case []byte:
			content = string(val)
		default:
			return nil, fmt.Errorf("unsupported %T", source)
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
