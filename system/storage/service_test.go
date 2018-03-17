package storage_test

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/system/exec"
	"github.com/viant/endly/system/storage"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	tstorage "github.com/viant/toolbox/storage"
	_ "github.com/viant/toolbox/storage/scp"
	"github.com/viant/toolbox/url"
	"io/ioutil"
	"path"
	"strings"
	"testing"
)

//update context for data substitution
func updateContext(context *endly.Context) {
	var state = context.State()
	endpointMap := data.NewMap()
	endpointMap.Put("host", "127.0.0.1")
	endpointMap.Put("port", "8080")
	state.Put("endpoint", endpointMap)
}

func SetupMemoryStorage() {
	memStorage := tstorage.NewMemoryService()
	parent := toolbox.CallerDirectory(3)
	fileContent, _ := ioutil.ReadFile(path.Join(parent, "test/config.json"))
	memStorage.Upload("mem:///test/copy/storage/config1.json", bytes.NewReader(fileContent))
	memStorage.Upload("mem:///test/copy/storage/config2.json", bytes.NewReader(fileContent))
	memStorage.Upload("mem:///test/copy/storage/data/data.json", strings.NewReader("{\"key\":1}"))
	memStorage.Upload("mem:///tmp/copy2_source/config1.json", bytes.NewReader(fileContent))
	memStorage.Upload("mem:///tmp/copy2_source/config2.json", bytes.NewReader(fileContent))
	memStorage.Upload("mem:///tmp/copy2_source/copy2_source.tar.gz", strings.NewReader("123"))
	memStorage.Upload("mem:///tmp/copy2_source/config1.json.tar.gz", strings.NewReader("abc"))
	memStorage.Upload("mem:///tmp/copy2_source/config2.json.tar.gz", strings.NewReader("xyz"))
}

func TestTransferService_Copy(t *testing.T) {
	var target = url.NewResource("ssh://127.0.0.1:22")
	memStorage := tstorage.NewMemoryService()
	SetupMemoryStorage()
	var manager = endly.New()
	var useCases = []struct {
		baseDir  string
		Request  *storage.CopyRequest
		Expected map[string]string
		Error    string
	}{
		{
			"test/copy/compress/dir/darwin",
			&storage.CopyRequest{
				Transfers: []*storage.Transfer{

					{
						Source: url.NewResource("scp://127.0.0.1:22/tmp/copy2_source"),
						Dest:   url.NewResource("/tmp/copy2_target"),
					},
				},
			},
			map[string]string{
				"mem:///tmp/copy2_target/copy2_source.tar.gz": "123",
			},
			"",
		},
		{
			"test/copy/compress/file1/darwin",
			&storage.CopyRequest{

				Transfers: []*storage.Transfer{
					{
						Compress: true,
						Source:   url.NewResource("scp://127.0.0.1:22/tmp/copy2_source/config1.json"),
						Dest:     url.NewResource("/tmp/copy3_target"),
					},
				},
			},
			map[string]string{
				"mem:///tmp/copy3_target/config1.json.tar.gz": "abc",
			},
			"",
		},
		{
			"test/copy/compress/file2/darwin",
			&storage.CopyRequest{
				Transfers: []*storage.Transfer{
					{
						Source:   url.NewResource("scp://127.0.0.1:22/tmp/copy2_source/config2.json"),
						Dest:     url.NewResource("/tmp/copy4_target/config2.json"),
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

		context, err := exec.NewSSHReplayContext(manager, target, useCase.baseDir)
		storage.UseMemoryService(context)
		updateContext(context)
		defer context.Close()

		service, err := context.Service(storage.ServiceID)

		assert.Nil(t, err)
		if assert.Nil(t, err) {
			serviceResponse := service.Run(context, useCase.Request)
			assert.Equal(t, useCase.Error, serviceResponse.Error)
			response, ok := serviceResponse.Response.(*storage.CopyResponse)
			if !ok {
				assert.Fail(t, fmt.Sprintf("process serviceResponse was empty  %T", serviceResponse.Response))
				continue
			}
			if !assert.NotNil(t, response) {
				return
			}
			if assert.True(t, len(response.TransferredURL) > 0) {
				for URL, expected := range useCase.Expected {
					object, err := memStorage.StorageObject(URL)
					if assert.Nil(t, err, URL) {
						reader, err := memStorage.Download(object)
						if assert.Nil(t, err) {
							defer reader.Close()
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

}

func TestTransferService_Remove(t *testing.T) {
	var manager = endly.New()
	storageService, err := manager.Service(storage.ServiceID)
	assert.Nil(t, err)
	context := manager.NewContext(toolbox.NewContext())
	memStorage := storage.UseMemoryService(context)
	memStorage.Upload("mem:///test/remove/storage/config1.json", strings.NewReader("abc"))

	object, _ := memStorage.StorageObject("mem:///test/remove/storage/config1.json")
	assert.NotNil(t, object)

	serviceResponse := storageService.Run(context, &storage.RemoveRequest{
		Assets: []*url.Resource{
			url.NewResource("mem:///test/remove/storage/config1.json"),
			url.NewResource("mem:///dummy"),
		},
	})
	if assert.Equal(t, serviceResponse.Error, "") {
		response, ok := serviceResponse.Response.(*storage.RemoveResponse)
		assert.EqualValues(t, 1, len(response.Removed))
		if ok {
			assert.EqualValues(t, response.Removed[0], "mem:///test/remove/storage/config1.json")
		}
	}
	object, _ = memStorage.StorageObject("mem:///test/remove/storage/config1.json")
	assert.Nil(t, object)

}

func TestTransferService_Download(t *testing.T) {
	var manager = endly.New()
	storageService, err := manager.Service(storage.ServiceID)
	assert.Nil(t, err)
	context := manager.NewContext(toolbox.NewContext())
	memStorage := storage.UseMemoryService(context)
	memStorage.Upload("mem:///test/download/storage/config1.json", strings.NewReader("abc"))

	serviceResponse := storageService.Run(context, &storage.DownloadRequest{
		DestKey: "key1",
		Source:  url.NewResource("mem:///test/download/storage/config1.json"),
	})

	if assert.Equal(t, serviceResponse.Error, "") {
		response, ok := serviceResponse.Response.(*storage.DownloadResponse)
		if ok {
			assert.EqualValues(t, response.Payload, "abc")
		}
		state := context.State()
		assert.Equal(t, state.GetString("key1"), "abc")
	}

}

func TestTransferService_Upload(t *testing.T) {
	var manager = endly.New()
	storageService, err := manager.Service(storage.ServiceID)
	assert.Nil(t, err)
	context := manager.NewContext(toolbox.NewContext())
	memStorage := storage.UseMemoryService(context)
	state := context.State()
	state.Put("key10", "XYZ")

	serviceResponse := storageService.Run(context, &storage.UploadRequest{
		SourceKey: "key10",
		Dest:      url.NewResource("mem:///test/storage/upload/config1.json"),
	})

	if assert.Equal(t, serviceResponse.Error, "") {
		response, ok := serviceResponse.Response.(*storage.UploadResponse)
		if ok {
			assert.EqualValues(t, 3, response.UploadSize)
			assert.EqualValues(t, "mem:///test/storage/upload/config1.json", response.UploadURL)

			object, err := memStorage.StorageObject("mem:///test/storage/upload/config1.json")
			assert.Nil(t, err)

			reader, err := memStorage.Download(object)
			assert.Nil(t, err)

			content, err := ioutil.ReadAll(reader)
			assert.Nil(t, err)
			assert.Equal(t, "XYZ", string(content))
		}
	}

}

func TestTransferService_Upload_Error(t *testing.T) {
	var manager = endly.New()
	storageService, err := manager.Service(storage.ServiceID)
	assert.Nil(t, err)
	context := manager.NewContext(toolbox.NewContext())

	serviceResponse := storageService.Run(context, &storage.UploadRequest{
		SourceKey: "key10",
		Dest:      url.NewResource("mem:///test/storage/upload/config1.json"),
	})
	assert.Equal(t, "sourcekey key10 value was empty at storage.upload", serviceResponse.Error)
}

func TestStorageCopyRequest_Validate(t *testing.T) {

	{
		request := storage.CopyRequest{}
		assert.NotNil(t, request.Validate())
	}
	{
		request := storage.CopyRequest{
			Transfers: []*storage.Transfer{},
		}
		assert.NotNil(t, request.Validate())
	}

	{
		request := storage.CopyRequest{
			Transfers: []*storage.Transfer{
				{
					Source: url.NewResource("abc"),
				},
			},
		}
		assert.NotNil(t, request.Validate())
	}
	{
		request := storage.CopyRequest{
			Transfers: []*storage.Transfer{
				{
					Dest: url.NewResource("abc"),
				},
			},
		}
		assert.NotNil(t, request.Validate())
	}

	{
		request := storage.CopyRequest{}
		assert.NotNil(t, request.Validate())
	}
	{
		request := storage.CopyRequest{
			Transfers: []*storage.Transfer{},
		}
		assert.NotNil(t, request.Validate())
	}

	{
		request := storage.CopyRequest{
			Transfers: []*storage.Transfer{
				{
					Source: url.NewResource("abc"),
				},
			},
		}
		assert.NotNil(t, request.Validate())
	}
	{
		request := storage.CopyRequest{
			Transfers: []*storage.Transfer{
				{
					Dest:   url.NewResource("abc"),
					Source: url.NewResource("xyz"),
				},
			},
		}
		assert.Nil(t, request.Validate())
	}

}

func TestStorageDownloadRequest_Validate(t *testing.T) {

	{
		request := storage.DownloadRequest{}
		assert.NotNil(t, request.Validate())
	}
	{
		request := storage.DownloadRequest{
			DestKey: "abc",
		}
		assert.NotNil(t, request.Validate())
	}
	{
		request := storage.DownloadRequest{
			Source: url.NewResource("abc"),
		}
		assert.NotNil(t, request.Validate())
	}
	{
		request := storage.DownloadRequest{
			Source:  url.NewResource("abc"),
			DestKey: "a",
		}
		assert.Nil(t, request.Validate())
	}
}

func TestStorageUploadRequest_Validate(t *testing.T) {

	{
		request := storage.UploadRequest{}
		assert.NotNil(t, request.Validate())
	}
	{
		request := storage.UploadRequest{
			SourceKey: "abc",
		}
		assert.NotNil(t, request.Validate())
	}
	{
		request := storage.UploadRequest{
			Dest: url.NewResource("abc"),
		}
		assert.NotNil(t, request.Validate())
	}
	{
		request := storage.UploadRequest{
			Dest:      url.NewResource("abc"),
			SourceKey: "a",
		}
		assert.Nil(t, request.Validate())
	}
}

func TestStorageUploadRemove_Validate(t *testing.T) {

	{
		request := storage.RemoveRequest{}
		assert.NotNil(t, request.Validate())
	}

	{
		request := storage.RemoveRequest{
			Assets: []*url.Resource{

				url.NewResource("a"),
			},
		}
		assert.Nil(t, request.Validate())
	}

}
