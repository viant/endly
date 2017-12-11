package endly_test

import (
	"testing"
	"github.com/viant/endly"
	"github.com/viant/endly/test/proto"
	"github.com/stretchr/testify/assert"
	"github.com/viant/toolbox"
)



func Test_AsProtobufMessage(t *testing.T) {
	var input = "{\"id\":1, \"name\":\"abc\"}"
	encoded, err := endly.AsProtobufMessage(input, nil, &proto.Message{})
	if assert.Nil(t, err) {
		assert.EqualValues(t, "base64:CAESA2FiYw==", encoded)
	}

	var inputMap = map[string]interface{}{
		"id":   1,
		"name": "abc",
	}
	encoded, err = endly.AsProtobufMessage(inputMap, nil, &proto.Message{})
	if assert.Nil(t, err) {
		assert.EqualValues(t, "base64:CAESA2FiYw==", encoded)
	}


	message, err := endly.FromProtobufMessage(encoded, nil,  &proto.Message{})
	if assert.Nil(t, err) {
		aMap := toolbox.AsMap(message)
		assert.EqualValues(t, 1, aMap["Id"])
		assert.EqualValues(t, "abc", aMap["Name"])
	}



}

func Test_AsProtobufMessage_Errors(t *testing.T) {

	{
		var input= "{id\":1, \"name\":\"abc\"}"
		_, err := endly.AsProtobufMessage(input, nil, &proto.Message{})
		assert.NotNil(t, err)
	}

	{
		var input= ""
		_, err := endly.AsProtobufMessage(input, nil, &proto.Message{})
		assert.NotNil(t, err)
	}
	{
		_, err := endly.FromProtobufMessage("base64:CAErSA2FiYw==", nil,  &proto.Message{})
		assert.NotNil(t, err)

	}
	{
		_, err := endly.FromProtobufMessage("base64:12=", nil,  &proto.Message{})
		assert.NotNil(t, err)

	}


}