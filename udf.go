package endly

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"strings"
)


//init initialises UDF functions
func init() {
	UdfRegistry["Dob"] = DateOfBirth
}


//TransformWithUDF transform payload with provided UDF name.
func TransformWithUDF(context *Context, udfName, source string, payload interface{}) (interface{}, error) {
	var state = context.State()
	var udf, has = UdfRegistry[udfName]
	if !has {
		if candidate, ok := state[udfName]; ok {
			udf, has = candidate.(func(source interface{}, state data.Map) (interface{}, error))
		}
	}
	if !has {
		return nil, fmt.Errorf("failed to lookup udf: %v for: %v", udfName, source)
	}
	transformed, err := udf(payload, state)
	if err != nil {
		return nil, fmt.Errorf("failed to run udf: %v, %v", udfName, err)
	}
	return transformed, nil
}

//DateOfBirth returns formated date of birth
func DateOfBirth(source interface{}, state data.Map) (interface{}, error) {
	if !toolbox.IsSlice(source) {
		return nil, fmt.Errorf("expected slice but had: %T %v", source, source)
	}
	return toolbox.NewDateOfBirthrovider().Get(toolbox.NewContext(), toolbox.AsSlice(source)...)
}

//AsProtobufMessage generic method for converting a map, or json string into a proto message
func AsProtobufMessage(source interface{}, state data.Map, target proto.Message) (interface{}, error) {
	var requestMap map[string]interface{}
	if toolbox.IsString(source) {
		requestMap = make(map[string]interface{})
		err := toolbox.NewJSONDecoderFactory().Create(strings.NewReader(toolbox.AsString(source))).Decode(&requestMap)
		if err != nil {
			fmt.Printf("failed to run udf: %v %v\n", source, err)
			return nil, err
		}
	} else {
		requestMap = toolbox.AsMap(source)
	}
	converter := toolbox.NewColumnConverter("")
	err := converter.AssignConverted(target, requestMap)
	if err != nil {
		fmt.Printf("failed to run udf: unable convert: %v %v\n", source, err)
		return nil, err
	}

	protodata, err := proto.Marshal(target)
	if err != nil {
		fmt.Printf("failed to run udf: unable Marshal %v %v\n", source, err)
		return nil, fmt.Errorf("failed to encode: %v, %v", requestMap, err)
	}
	buf := new(bytes.Buffer)
	encoder := base64.NewEncoder(base64.StdEncoding, buf)
	encoder.Write(protodata)
	err = encoder.Close()
	return fmt.Sprintf("base64:%v", string(buf.Bytes())), err
}

//FromProtobufMessage generic method for converting a proto message into a map
func FromProtobufMessage(source interface{}, state data.Map, sourceMessage proto.Message) (interface{}, error) {
	if toolbox.IsString(source) {
		textSource := toolbox.AsString(source)

		payload, err := FromPayload(textSource)
		if err != nil {
			return nil, err
		}
		converter := toolbox.NewColumnConverter("")
		err = proto.Unmarshal(payload, sourceMessage)
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
	return nil, fmt.Errorf("expected string but had:%T", source)
}
