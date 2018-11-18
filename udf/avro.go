package udf

import (
	"bytes"
	"fmt"
	"github.com/linkedin/goavro"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
	"io"
	"strings"
)

//NewAvroReader creates a new avro reader UDFs
func NewAvroReader(source interface{}, state data.Map) (interface{}, error) {
	var reader io.Reader
	switch data := source.(type) {
	case []byte:
		reader = bytes.NewReader(data)
	case string:
		reader = strings.NewReader(data)
	default:
		return nil, fmt.Errorf("unsupported input: %T, expected []byte or string", source)
	}
	avroReader, err := goavro.NewOCFReader(reader)
	if err != nil {
		return nil, err
	}
	var datums []interface{}
	for avroReader.Scan() {
		if datum, err := avroReader.Read(); err == nil {
			datums = append(datums, datum)
		}
	}
	if len(datums) == 1 {
		return toolbox.AsJSONText(datums[0])
	}
	return toolbox.AsJSONText(datums)
}

func getAvroSchema(args interface{}) (string, error) {
	textArg := strings.TrimSpace(toolbox.AsString(args))
	if strings.HasPrefix(textArg, "{") {
		return textArg, nil
	} else {
		resource := url.NewResource(textArg)
		return resource.DownloadText()
	}
}

//NewAvroWriter creates a new avro writer provider
func NewAvroWriter(args ...interface{}) (func(source interface{}, state data.Map) (interface{}, error), error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no sufficent args |usage: NewAvroWriter(avroSchema|URL, compression)")
	}
	schema, err := getAvroSchema(args[0])
	if err != nil {
		return nil, err
	}
	var compression = ""
	if len(args) > 1 {
		compression = toolbox.AsString(args[1])
	}
	return func(source interface{}, state data.Map) (interface{}, error) {
		writer := new(bytes.Buffer)
		avroWriter, err := goavro.NewOCFWriter(goavro.OCFConfig{
			W:               writer,
			Schema:          schema,
			CompressionName: compression,
		})
		if err != nil {
			return nil, err
		}
		var input interface{}
		switch source.(type) {
		case []byte, string:
			input, err = toolbox.JSONToInterface(source)
			if err != nil {
				return nil, fmt.Errorf("failed to convert %v to JSON: %v ", source, err)
			}
		default:
			input = toolbox.AsMap(source)
		}

		err = avroWriter.Append([]interface{}{input})
		return writer.Bytes(), err
	}, nil
}
