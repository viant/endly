package buildin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/viant/toolbox/data"
)

// LoadJSONLines loads a stream of JSON objects as a collection for zero, one or
// many records. Unlike LoadJSON it never collapses one object to a map. Records
// are decoded without a scanner's per-line size limit; malformed or non-object
// records reject the entire input rather than yielding a partial capture.
func LoadJSONLines(source interface{}, state data.Map) (interface{}, error) {
	filename, ok := source.(string)
	if !ok || strings.TrimSpace(filename) == "" {
		return nil, fmt.Errorf("LoadJSONLines requires a source filename")
	}
	var reader io.Reader
	if file, err := os.Open(filename); err == nil {
		// Read literal local paths first, including '#' and '?' characters that
		// must not be interpreted as URL fragments or queries.
		defer file.Close()
		reader = file
	} else {
		payload, err := LoadBinary(filename, state)
		if err != nil {
			return nil, err
		}
		switch value := payload.(type) {
		case []byte:
			reader = bytes.NewReader(value)
		case string:
			reader = strings.NewReader(value)
		default:
			return nil, fmt.Errorf("unsupported JSON Lines content type %T", payload)
		}
	}
	decoder := json.NewDecoder(reader)
	records := make([]interface{}, 0)
	for index := 1; ; index++ {
		var record map[string]interface{}
		if err := decoder.Decode(&record); err != nil {
			if err == io.EOF {
				return records, nil
			}
			return nil, fmt.Errorf("JSON Lines record %d: %w", index, err)
		}
		if record == nil {
			return nil, fmt.Errorf("JSON Lines record %d must be an object", index)
		}
		records = append(records, record)
	}
}
