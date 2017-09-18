package endly_test

import (
	"testing"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly/common"
)

type TestConfig struct {
	Endpoint struct {
		Hostname string
		Port string
	}
}

func TestNewTransferService(t *testing.T) {

	manager := endly.NewManager()
	service, err := manager.Service(endly.TransferServiceId)
	if assert.Nil(t, err) {


		context := manager.NewContext(toolbox.NewContext())

		endpointMap := common.NewMap()
		endpointMap.Put("host", "127.0.0.1")
		endpointMap.Put("port", "8080")
		var state = context.State()
		state.Put("endpoint", endpointMap)

		{
			response := service.Run(context, &endly.TransfersRequest{
				Transfers: []*endly.Transfer{
					{
						Source:   endly.NewFileResource("test/transfer/config.json"),
						Target:   endly.NewFileResource("/tmp/transfered.json"),
						Parsable: true,
					},
				},
			})
			assert.Equal(t, "", response.Error)
			serviceResponse, ok := response.Response.(*endly.TransfersResponse)
			assert.True(t, ok)
			assert.Equal(t, 1, len(serviceResponse.Transfered))
			assert.Equal(t, "file:///tmp/transfered.json", serviceResponse.Transfered[0].Target)

			reader, _, err := toolbox.OpenReaderFromURL("file:///tmp/transfered.json")
			assert.Nil(t, err)
			config := &TestConfig{}
			err = toolbox.NewJSONDecoderFactory().Create(reader).Decode(config)
			assert.Nil(t, err)
			assert.Equal(t, "127.0.0.1", config.Endpoint.Hostname)
			assert.Equal(t, "8080", config.Endpoint.Port)
		}
		{
			response := service.Run(context, &endly.TransfersRequest{
				Transfers: []*endly.Transfer{
					{
						Source:   endly.NewFileResource("test/transfer/config.json"),
						Target:   endly.NewFileResource("/tmp/transfered.json"),
					},
				},
			})
			assert.Equal(t, "", response.Error)
			serviceResponse, ok := response.Response.(*endly.TransfersResponse)
			assert.True(t, ok)
			assert.Equal(t, 1, len(serviceResponse.Transfered))
			assert.Equal(t, "file:///tmp/transfered.json", serviceResponse.Transfered[0].Target)

			reader, _, err := toolbox.OpenReaderFromURL("file:///tmp/transfered.json")
			assert.Nil(t, err)
			config := &TestConfig{}
			err = toolbox.NewJSONDecoderFactory().Create(reader).Decode(config)
			assert.Nil(t, err)
			assert.Equal(t, "$endpoint.host", config.Endpoint.Hostname)
			assert.Equal(t, "$endpoint.port", config.Endpoint.Port)
		}
		err = toolbox.RemoveFileIfExist("/tmp/transfered.json")
		assert.Nil(t, err)

	}

}
