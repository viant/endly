package endly

import (
	"github.com/golang/protobuf/proto"
	"github.com/viant/toolbox"
	"fmt"
	"bytes"
	"encoding/base64"
	"strings"
	"io/ioutil"
	"github.com/viant/endly/common"
)

//AsProtobufMessage generic method for converting a map into a proto message
func AsProtobufMessage(source interface{}, state common.Map, target proto.Message) (interface{}, error) {
	var requestMap = toolbox.AsMap(source)
	converter := toolbox.NewColumnConverter("")
	err :=converter.AssignConverted(target, requestMap)
	if err != nil {
		return nil, err
	}

	protodata, err := proto.Marshal(target)
	if err != nil {
		return nil, fmt.Errorf("Failed to encode: %v, %v", requestMap, err)
	}
	buf := new(bytes.Buffer)
	encoder := base64.NewEncoder(base64.StdEncoding, buf)
	encoder.Write(protodata)
	encoder.Close()
	return fmt.Sprintf("base64:%v", string(buf.Bytes())), nil
}



//AsProtobufMessage generic method for converting a proto message into a map
func FromProtobufMessage(source interface{}, state common.Map, sourceMessage proto.Message) (interface{}, error) {
	if toolbox.IsString(source) {
		textSource := toolbox.AsString(source);
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
			err =converter.AssignConverted(&resultMap, sourceMessage)
			if err != nil {
				return nil, err
			}
			return toolbox.DereferenceValues(resultMap), nil
		}

	}
	return nil, fmt.Errorf("Expected string but had:%T", source)
}


