package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
	"testing"
)

type TestConfig struct {
	Endpoint struct {
		Hostname string
		Port     string
	}
}

func TestNewTransferService(t *testing.T) {

	manager := endly.NewManager()
	service, err := manager.Service(endly.TransferServiceID)
	if assert.Nil(t, err) {

		context := manager.NewContext(toolbox.NewContext())

		endpointMap := data.NewMap()
		endpointMap.Put("host", "127.0.0.1")
		endpointMap.Put("port", "8080")
		var state = context.State()
		state.Put("endpoint", endpointMap)

		{
			response := service.Run(context, &endly.TransferCopyRequest{
				Transfers: []*endly.Transfer{
					{
						Source: url.NewResource("test/transfer/config.json"),
						Target: url.NewResource("/tmp/transfered.json"),
						Expand: true,
					},
				},
			})
			assert.Equal(t, "", response.Error)
			serviceResponse, ok := response.Response.(*endly.TransferCopyResponse)
			assert.True(t, ok)
			assert.Equal(t, 1, len(serviceResponse.Transferred))
			assert.Equal(t, "file:///tmp/transfered.json", serviceResponse.Transferred[0].TargetURL)

			reader, _, err := toolbox.OpenReaderFromURL("file:///tmp/transfered.json")
			assert.Nil(t, err)
			defer reader.Close()
			config := &TestConfig{}
			err = toolbox.NewJSONDecoderFactory().Create(reader).Decode(config)
			assert.Nil(t, err)
			assert.Equal(t, "127.0.0.1", config.Endpoint.Hostname)
			assert.Equal(t, "8080", config.Endpoint.Port)
		}
		{
			response := service.Run(context, &endly.TransferCopyRequest{
				Transfers: []*endly.Transfer{
					{
						Source: url.NewResource("test/transfer/config.json"),
						Target: url.NewResource("/tmp/transfered.json"),
					},
				},
			})
			assert.Equal(t, "", response.Error)
			serviceResponse, ok := response.Response.(*endly.TransferCopyResponse)
			assert.True(t, ok)
			assert.Equal(t, 1, len(serviceResponse.Transferred))
			assert.Equal(t, "file:///tmp/transfered.json", serviceResponse.Transferred[0].TargetURL)

			reader, _, err := toolbox.OpenReaderFromURL("file:///tmp/transfered.json")
			assert.Nil(t, err)
			reader.Close()
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
