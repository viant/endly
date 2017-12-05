package endly_test

import (
	"path"
	"os"
	"github.com/viant/endly"
	"github.com/stretchr/testify/assert"
	"testing"
	"github.com/viant/toolbox/url"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/storage"
	"github.com/viant/toolbox"
	"io/ioutil"
	"bytes"
	"strings"
)

//import (
//	"github.com/stretchr/testify/assert"
//	"github.com/viant/endly"
//	"github.com/viant/toolbox"
//	"github.com/viant/toolbox/data"
//	"github.com/viant/toolbox/url"
//	"testing"
//)
//
//type TestConfig struct {
//	Endpoint struct {
//		Hostname string
//		Port     string
//	}
//}

func TestTransferService_Transffer(t *testing.T) {


	parent := toolbox.CallerDirectory(3)
	fileContent, err := ioutil.ReadFile(path.Join(parent, "test/transfer/config.json"))
	if ! assert.Nil(t, err) {
		return
	}

	var credentialFile = path.Join(os.Getenv("HOME"), ".secret/scp.json")

	//var target = url.NewResource("scp://35.197.115.53:22/", credentialFile) //
	var target = url.NewResource("scp://127.0.0.1:22/", credentialFile) //
	manager := endly.NewManager()


	context, err := OpenTestRecorderContext(manager, target, "test/transfer/copy/expand/darwin")
	///context := manager.NewContext(toolbox.NewContext())

	defer context.Close()

	systemService, err := context.Service(endly.TransferServiceID)
	assert.Nil(t, err)

	memStorage := storage.NewMemoryService()
	memStorage.Upload("mem:///test/transfer/config1.json", bytes.NewReader(fileContent))
	memStorage.Upload("mem:///test/transfer/config2.json", bytes.NewReader(fileContent))
	memStorage.Upload("mem:///test/transfer/data/data.json", strings.NewReader("{\"key\":1}"))


	endpointMap := data.NewMap()
	endpointMap.Put("host", "127.0.0.1")
	endpointMap.Put("port", "8080")
	var state = context.State()
	state.Put("endpoint", endpointMap)
	serviceResponse := systemService.Run(context, &endly.TransferCopyRequest{
		Transfers: []*endly.Transfer{
			{
				Source: url.NewResource("mem:///test/transfer/"),
				Target: url.NewResource("mem:///tmp/"),
				Expand: true,
			},
		},
	})
	assert.Equal(t, "", serviceResponse.Error)
	response, ok := serviceResponse.Response.(*endly.TransferCopyResponse)
	if assert.True(t, ok) && serviceResponse != nil {
		assert.True(t, len(response.Transferred) > 0)
	}

}

//func TestNewTransferService(t *testing.T) {
//
//	manager := endly.NewManager()
//	service, err := manager.Service(endly.TransferServiceID)
//	if assert.Nil(t, err) {
//
//		context := manager.NewContext(toolbox.NewContext())
//
//		endpointMap := data.NewMap()
//		endpointMap.Put("host", "127.0.0.1")
//		endpointMap.Put("port", "8080")
//		var state = context.State()
//		state.Put("endpoint", endpointMap)
//
//		{
//			response := service.Run(context, &endly.TransferCopyRequest{
//				Transfers: []*endly.Transfer{
//					{
//						Source: url.NewResource("test/transfer/config.json"),
//						Target: url.NewResource("/tmp/transfered.json"),
//						Expand: true,
//					},
//				},
//			})
//			assert.Equal(t, "", response.Error)
//			serviceResponse, ok := response.Response.(*endly.TransferCopyResponse)
//			assert.True(t, ok)
//			assert.Equal(t, 1, len(serviceResponse.Transferred))
//			assert.Equal(t, "file:///tmp/transfered.json", serviceResponse.Transferred[0].TargetURL)
//
//			reader, _, err := toolbox.OpenReaderFromURL("file:///tmp/transfered.json")
//			assert.Nil(t, err)
//			defer reader.Close()
//			config := &TestConfig{}
//			err = toolbox.NewJSONDecoderFactory().Create(reader).Decode(config)
//			assert.Nil(t, err)
//			assert.Equal(t, "127.0.0.1", config.Endpoint.Hostname)
//			assert.Equal(t, "8080", config.Endpoint.Port)
//		}
//		{
//			response := service.Run(context, &endly.TransferCopyRequest{
//				Transfers: []*endly.Transfer{
//					{
//						Source: url.NewResource("test/transfer/config.json"),
//						Target: url.NewResource("/tmp/transfered.json"),
//					},
//				},
//			})
//			assert.Equal(t, "", response.Error)
//			serviceResponse, ok := response.Response.(*endly.TransferCopyResponse)
//			assert.True(t, ok)
//			assert.Equal(t, 1, len(serviceResponse.Transferred))
//			assert.Equal(t, "file:///tmp/transfered.json", serviceResponse.Transferred[0].TargetURL)
//
//			reader, _, err := toolbox.OpenReaderFromURL("file:///tmp/transfered.json")
//			assert.Nil(t, err)
//			reader.Close()
//			config := &TestConfig{}
//			err = toolbox.NewJSONDecoderFactory().Create(reader).Decode(config)
//			assert.Nil(t, err)
//			assert.Equal(t, "$endpoint.host", config.Endpoint.Hostname)
//			assert.Equal(t, "$endpoint.port", config.Endpoint.Port)
//		}
//		err = toolbox.RemoveFileIfExist("/tmp/transfered.json")
//		assert.Nil(t, err)
//
//	}
//
//}
