package endly_test

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/storage"
	"github.com/viant/toolbox/url"
	"io/ioutil"
	"path"
	"strings"
	"testing"
)

func updateContext(context *endly.Context) {
	var state = context.State()

	endpointMap := data.NewMap()
	endpointMap.Put("host", "127.0.0.1")
	endpointMap.Put("port", "8080")

	state.Put("endpoint", endpointMap)
}

func TestTransferService_Copy(t *testing.T) {

	var target = url.NewResource("ssh://127.0.0.1:22")

	parent := toolbox.CallerDirectory(3)
	fileContent, err := ioutil.ReadFile(path.Join(parent, "test/transfer/config.json"))
	if !assert.Nil(t, err) {
		return
	}
	memStorage := storage.NewMemoryService()
	memStorage.Upload("mem:///test/copy/transfer/config1.json", bytes.NewReader(fileContent))
	memStorage.Upload("mem:///test/copy/transfer/config2.json", bytes.NewReader(fileContent))
	memStorage.Upload("mem:///test/copy/transfer/data/data.json", strings.NewReader("{\"key\":1}"))

	memStorage.Upload("mem:///tmp/copy2_source/config1.json", bytes.NewReader(fileContent))
	memStorage.Upload("mem:///tmp/copy2_source/config2.json", bytes.NewReader(fileContent))

	memStorage.Upload("mem:///tmp/copy2_source/copy2_source.tar.gz", strings.NewReader("123"))
	memStorage.Upload("mem:///tmp/copy2_source/config1.json.tar.gz", strings.NewReader("abc"))
	memStorage.Upload("mem:///tmp/copy2_source/config2.json.tar.gz", strings.NewReader("xyz"))

	var manager = endly.NewManager()
	var useCases = []struct {
		baseDir  string
		Request  *endly.TransferCopyRequest
		Expected map[string]string
		Error    string
	}{
		{
			"",
			&endly.TransferCopyRequest{
				[]*endly.Transfer{
					{
						Source: url.NewResource("mem:///test/copy/transfer/"),
						Target: url.NewResource("mem:///tmp/copy1"),
						Expand: true,
					},
				},
			},
			map[string]string{
				"mem:///tmp/copy1/config1.json": "{\n  \"Endpoint\": {\n    \"Hostname\":\"127.0.0.1\",\n    \"Port\":\"8080\"\n  }\n}",
			},
			"",
		},
		{
			"",
			&endly.TransferCopyRequest{
				[]*endly.Transfer{
					{
						Source: url.NewResource("mem:///test/copy/transfer/"),
						Target: url.NewResource("mem:///tmp/copy1"),
						Expand: false,
					},
				},
			},
			map[string]string{
				"mem:///tmp/copy1/config1.json": "{\n  \"Endpoint\": {\n    \"Hostname\":\"$endpoint.host\",\n    \"Port\":\"$endpoint.port\"\n  }\n}",
			},
			"",
		},
		{
			"test/transfer/copy/compress/dir/darwin",
			&endly.TransferCopyRequest{
				[]*endly.Transfer{
					{
						Source:   url.NewResource("scp://127.0.0.1:22/tmp/copy2_source"),
						Target:   url.NewResource("/tmp/copy2_target"),
						Compress: true,
					},
				},
			},
			map[string]string{
				"mem:///tmp/copy2_target/copy2_source.tar.gz": "123",
			},
			"",
		},
		{
			"test/transfer/copy/compress/file1/darwin",
			&endly.TransferCopyRequest{
				[]*endly.Transfer{
					{
						Source:   url.NewResource("scp://127.0.0.1:22/tmp/copy2_source/config1.json"),
						Target:   url.NewResource("/tmp/copy3_target"),
						Compress: true,
					},
				},
			},
			map[string]string{
				"mem:///tmp/copy3_target/config1.json.tar.gz": "abc",
			},
			"",
		},
		{
			"test/transfer/copy/compress/file2/darwin",
			&endly.TransferCopyRequest{
				[]*endly.Transfer{
					{
						Source:   url.NewResource("scp://127.0.0.1:22/tmp/copy2_source/config2.json"),
						Target:   url.NewResource("/tmp/copy4_target/config2.json"),
						Compress: true,
					},
				},
			},
			map[string]string{
				"mem:///tmp/copy4_target/config2.json.tar.gz": "xyz",
			},
			"",
		},
	}

	for _, useCase := range useCases {
		context := manager.NewContext(toolbox.NewContext())

		if useCase.baseDir != "" {
			execService, err := GetReplayService(useCase.baseDir)
			if assert.Nil(t, err) {
				context, err = OpenTestContext(manager, target, execService)
				if !assert.Nil(t, err) {
					continue
				}
				var state = context.State()
				state.Put(endly.UseMemoryService, true)
			}
		}

		updateContext(context)
		defer context.Close()
		service, err := context.Service(endly.TransferServiceID)

		assert.Nil(t, err)
		if assert.Nil(t, err) {
			serviceResponse := service.Run(context, useCase.Request)
			assert.Equal(t, useCase.Error, serviceResponse.Error)
			response, ok := serviceResponse.Response.(*endly.TransferCopyResponse)
			if !ok {
				assert.Fail(t, fmt.Sprintf("process serviceResponse was empty  %T", serviceResponse.Response))
				continue
			}
			assert.True(t, len(response.Transferred) > 0)
			for URL, expected := range useCase.Expected {
				object, err := memStorage.StorageObject(URL)
				if assert.Nil(t, err, URL) {
					reader, err := memStorage.Download(object)
					if assert.Nil(t, err) {
						content, err := ioutil.ReadAll(reader)
						if assert.Nil(t, err) {
							assert.EqualValues(t, expected, string(content), URL)
						}
					}
				}
			}
		}
	}

}
