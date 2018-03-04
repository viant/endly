package log

import (
	"github.com/viant/toolbox"
	"strings"
)

//Record represents a log record
type Record struct {
	URL    string
	Number int
	Line   string
}

//IndexedRecord represents indexed log record
type IndexedRecord struct {
	*Record
	IndexValue string
}

//AsMap returns log records as map
func (r *Record) AsMap() (map[string]interface{}, error) {
	var result = make(map[string]interface{})
	err := toolbox.NewJSONDecoderFactory().Create(strings.NewReader(r.Line)).Decode(&result)
	return result, err
}
