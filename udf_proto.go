package endly

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"io/ioutil"
	"strings"
)

//AsProtobufMessage generic method for converting a map, or json string into a proto message
func AsProtobufMessage(source interface{}, state data.Map, target proto.Message) (interface{}, error) {
	var requestMap map[string]interface{}
	if toolbox.IsString(source) {
		requestMap = make(map[string]interface{})
		err := toolbox.NewJSONDecoderFactory().Create(strings.NewReader(toolbox.AsString(source))).Decode(&requestMap)
		if err != nil {
			fmt.Printf("Failed to run udf: %v %v\n", source, err)
			return nil, err
		}
	} else {
		requestMap = toolbox.AsMap(source)
	}
	converter := toolbox.NewColumnConverter("")
	err := converter.AssignConverted(target, requestMap)
	if err != nil {
		fmt.Printf("Failed to run udf: unable convert: %v %v\n", source, err)
		return nil, err
	}

	protodata, err := proto.Marshal(target)
	if err != nil {
		fmt.Printf("Failed to run udf: unable Marshal %v %v\n", source, err)
		return nil, fmt.Errorf("Failed to encode: %v, %v", requestMap, err)
	}
	buf := new(bytes.Buffer)
	encoder := base64.NewEncoder(base64.StdEncoding, buf)
	defer encoder.Close()
	encoder.Write(protodata)

	return fmt.Sprintf("base64:%v", string(buf.Bytes())), nil
}

//FromProtobufMessage generic method for converting a proto message into a map
func FromProtobufMessage(source interface{}, state data.Map, sourceMessage proto.Message) (interface{}, error) {
	if toolbox.IsString(source) {
		textSource := toolbox.AsString(source)
		if strings.HasPrefix(textSource, "base64:") {
			textSource = string(textSource[7:])
			decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(textSource))
			data, err := ioutil.ReadAll(decoder)
			if err != nil {
				return nil, err
			}

			converter := toolbox.NewColumnConverter("")

			err = proto.Unmarshal(data, sourceMessage)
			if err != nil {
				return nil, err
			}

			var resultMap = make(map[string]interface{})
			err = converter.AssignConverted(&resultMap, sourceMessage)
			if err != nil {
				return nil, err
			}
			return toolbox.DereferenceValues(resultMap), nil
		}

	}
	return nil, fmt.Errorf("ExpectedLogRecords string but had:%T", source)
}
