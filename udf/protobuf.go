package udf

import (
	"bytes"
	"fmt"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/dynamic/msgregistry"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"io"
	"io/ioutil"
	"path"
	"strings"
)

//ProtoCodec represent a proto codec
type ProtoCodec struct {
	registry *msgregistry.MessageRegistry
	msgType  string
}

func (c *ProtoCodec) AsMessage(msgType string, data []byte) (interface{}, error) {
	msgDescriptor, err := c.registry.FindMessageTypeByUrl(msgType)
	if err != nil {
		return nil, err
	}
	protoMsg := dynamic.NewMessage(msgDescriptor)
	if err = protoMsg.Unmarshal(data); err != nil {
		return nil, err
	}
	JSON, err := protoMsg.MarshalJSON()
	if err != nil {
		return nil, err
	}
	var result = make(map[string]interface{})
	err = toolbox.NewJSONDecoderFactory().Create(bytes.NewReader(JSON)).Decode(&result)
	return result, err
}

func (c *ProtoCodec) AsBinary(msgType string, msg interface{}) ([]byte, error) {
	msgDescriptor, err := c.registry.FindMessageTypeByUrl(msgType)
	if err != nil {
		return nil, err
	}
	var reader io.Reader
	switch value := msg.(type) {
	case string:
		reader = strings.NewReader(value)
	case []byte:
		reader = bytes.NewReader(value)
	default:
		text, err := toolbox.AsJSONText(msg)
		if err != nil {
			return nil, err
		}
		reader = strings.NewReader(text)
	}
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	protoMsg := dynamic.NewMessage(msgDescriptor)
	if err = protoMsg.UnmarshalJSON(data); err != nil {
		return nil, err
	}
	return protoMsg.Marshal()
}

//NewProtoCodec creates a new protobuf codec
func NewProtoCodec(schemaFile, importPath string, msgType string) (*ProtoCodec, error) {
	parser := protoparse.Parser{ImportPaths: []string{importPath}, IncludeSourceCodeInfo: true}
	descriptors, err := parser.ParseFiles(schemaFile)
	if err != nil {
		return nil, err
	}
	baseURL := ""
	registry := msgregistry.NewMessageRegistryWithDefaults()
	for _, desc := range descriptors {
		registry.AddFile(baseURL, desc)
	}
	return &ProtoCodec{
		registry: registry,
		msgType:  msgType,
	}, nil

}

func getProtoCodec(source string, args []interface{}) (*ProtoCodec, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("no sufficent args |usage: %v(schemaFile, messageType, importPath)", source)
	}
	schemaFile := toolbox.AsString(args[0])
	messageType := toolbox.AsString(args[1])
	importPath, filename := path.Split(schemaFile)
	if len(args) > 2 {
		importPath = toolbox.AsString(args[2])
	} else {
		schemaFile = filename
	}
	return NewProtoCodec(schemaFile, importPath, messageType)
}

//NewProtoWriter creates a new proto writer provider
func NewProtoWriter(args ...interface{}) (func(source interface{}, state data.Map) (interface{}, error), error) {
	codec, err := getProtoCodec("NewProtoWriter", args)
	if err != nil {
		return nil, err
	}
	return func(source interface{}, state data.Map) (interface{}, error) {
		return codec.AsBinary(codec.msgType, source)
	}, nil
}

//NewProtoWriter creates a new proto writer provider
func NewProtoReader(args ...interface{}) (func(source interface{}, state data.Map) (interface{}, error), error) {
	codec, err := getProtoCodec("NewProtoReader", args)
	if err != nil {
		return nil, err
	}
	return func(source interface{}, state data.Map) (interface{}, error) {
		var reader io.Reader
		switch val := source.(type) {
		case []byte:
			reader = bytes.NewReader(val)
		case string:
			reader = strings.NewReader(val)
		case io.Reader:
			reader = val
		}
		binaryData, err := ioutil.ReadAll(reader)
		if err != nil {
			return nil, err
		}
		return codec.AsMessage(codec.msgType, binaryData)
	}, nil
}
