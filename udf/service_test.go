package udf

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"io/ioutil"
	"log"
	"path"
	"testing"
)

func Test_Register(t *testing.T) {

	parentDirectory := toolbox.CallerDirectory(3)
	//
	//description:"message with package",
	//	importPath: path.Join(parentDirectory, "test/proto/pkg"),
	//		protoFile:"person.proto",
	//		messageType:"foo.Person",
	//		dataFile:path.Join(parentDirectory, "test/proto/pkg/data.json"),
	//

	var useCases = []struct {
		description string
		register    *RegisterRequest
		requestURL  string
		dataFile    string
		hasError    bool
	}{
		{
			description: "proto udf registration",
			register: &RegisterRequest{
				UDFs: []*endly.UdfProvider{
					{
						ID:       "myProtoUdf",
						Provider: "ProtoWriter",
						Params: []interface{}{
							"person.proto",
							"foo.Person",
							path.Join(parentDirectory, "test/proto/pkg"),
						},
					},
				},
			},
			dataFile: path.Join(parentDirectory, "test/proto/pkg/data.json"),
		},
		{
			description: "proto udf registration",
			requestURL:  path.Join("test/req.yaml"),
			dataFile:    path.Join(parentDirectory, "test/proto/book/data.json"),
		},
		{
			description: "proto udf error registration  - invalid parametrs",
			register: &RegisterRequest{
				UDFs: []*endly.UdfProvider{
					{
						ID:       "myProtoUdf",
						Provider: "ProtoWriter",
						Params: []interface{}{
							"person.proto",
							"foo.Person",
							"abc",
						},
					},
				},
			},
			hasError: true,
		},
		{
			description: "unknown provider error",
			register: &RegisterRequest{
				UDFs: []*endly.UdfProvider{
					{
						ID:       "myProtoUdf",
						Provider: "ABCProvider",
					},
				},
			},
			hasError: true,
		},
	}

	for _, useCase := range useCases {
		context := endly.New().NewContext(nil)
		state := context.State()
		state.Put("pd", parentDirectory)
		if useCase.requestURL != "" {
			var err error
			useCase.register, err = NewRegisterRequestFromURL(useCase.requestURL)
			if !assert.Nil(t, err, useCase.description) {
				continue
			}
		}

		err := endly.Run(context, useCase.register, nil)
		if useCase.hasError {
			assert.NotNil(t, err, useCase.description)
			continue
		}

		if !assert.Nil(t, err, useCase.description) {
			log.Print(err)
			continue
		}

		data, err := ioutil.ReadFile(useCase.dataFile)
		if !assert.Nil(t, err, useCase.description) {
			continue
		}

		udf := endly.UdfRegistry[useCase.register.UDFs[0].ID]
		converted, err := udf(data, context.State())
		assert.Nil(t, err, useCase.description)
		assert.NotNil(t, converted)
	}

}
