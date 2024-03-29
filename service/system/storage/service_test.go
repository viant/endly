package storage_test

/*

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/service/system/exec"
	"github.com/viant/endly/service/system/storage"
	"github.com/viant/neatly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	tstorage "github.com/viant/toolbox/storage"
	_ "github.com/viant/toolbox/storage/scp"
	"github.com/viant/endly/model/location"
)

//update context for data substitution
func updateContext(context *endly.Context) {
	var state = context.State()
	endpointMap := data.NewMap()
	endpointMap.Put("host", "127.0.0.1")
	endpointMap.Put("port", "8080")
	state.Put("endpoint", endpointMap)
	neatly.AddStandardUdf(state)
}

func SetupMemoryStorage() {
	memStorage := tstorage.NewMemoryService()
	parent := toolbox.CallerDirectory(3)
	fileContent, _ := ioutil.ReadFile(path.Join(parent, "test/config.json"))
	memStorage.Upload("mem:///test/Copy/storage/config1.json", bytes.NewReader(fileContent))
	memStorage.Upload("mem:///test/Copy/storage/config2.json", bytes.NewReader(fileContent))
	memStorage.Upload("mem:///test/Copy/storage/data/data.json", strings.NewReader("{\"key\":1}"))
	memStorage.Upload("mem:///tmp/copy2_source/config1.json", bytes.NewReader(fileContent))
	memStorage.Upload("mem:///tmp/copy2_source/config2.json", bytes.NewReader(fileContent))
	memStorage.Upload("mem:///tmp/copy5_source/config5.json", strings.NewReader("compressed"))
	memStorage.Upload("mem:///tmp/copy2_source/copy2_source.tar.gz", strings.NewReader("123"))
	memStorage.Upload("mem:///tmp/copy2_source/config1.json.tar.gz", strings.NewReader("abc"))
	memStorage.Upload("mem:///tmp/copy2_source/config2.json.tar.gz", strings.NewReader("xyz"))
}

func TestTransferService_Copy(t *testing.T) {
	var target = location.NewResource("ssh://127.0.0.1:22")
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
			"test/Copy/compress/dir/darwin",
			&storage.CopyRequest{
				Transfers: []*storage.Transfer{

					{
						Source: location.NewResource("scp://127.0.0.1:22/tmp/copy2_source"),
						Dest:   location.NewResource("/tmp/copy2_target"),
					},
				},
			},
			map[string]string{
				"mem:///tmp/copy2_target/copy2_source.tar.gz": "123",
			},
			"",
		},
		{
			"test/Copy/compress/file1/darwin",
			&storage.CopyRequest{

				Transfers: []*storage.Transfer{
					{
						Compress: true,
						Source:   location.NewResource("scp://127.0.0.1:22/tmp/copy2_source/config1.json"),
						Dest:     location.NewResource("/tmp/copy3_target"),
					},
				},
			},
			map[string]string{
				"mem:///tmp/copy3_target/config1.json.tar.gz": "abc",
			},
			"",
		},
		{
			"test/Copy/compress/file2/darwin",
			&storage.CopyRequest{
				Transfers: []*storage.Transfer{
					{
						Source:   location.NewResource("scp://127.0.0.1:22/tmp/copy2_source/config2.json"),
						Dest:     location.NewResource("/tmp/copy4_target/config4.json"),
						Compress: true,
					},
				},
			},
			map[string]string{
				"mem:///tmp/copy4_target/config4.json.tar.gz": "xyz",
			},
			"",
		},
		{
			"test/Copy/compress/file2/darwin",
			&storage.CopyRequest{
				Transfers: []*storage.Transfer{
					{
						Source:   location.NewResource("scp://127.0.0.1:22/tmp/copy5_source/config5.json"),
						Dest:     location.NewResource("/tmp/copy5_target/config5.gz"),
						Compress: false,
					},
				},
				Udf: "CopyWithCompression",
			},
			map[string]string{
				"mem:///tmp/copy5_target/config5.gz": "\x1f\x8b\b\x00\x00\tn\x88\x04\xff\x00\n\x00\xf5\xffcompressed\x00\x00\x00\xff\xff\x00\x00\x00\xff\xff\x01\x00\x00\xff\xff\x1eKV\x97\n\x00\x00\x00",
			},
			"",
		},
		{
			"test/Copy/compress/file2/darwin",
			&storage.CopyRequest{
				Transfers: []*storage.Transfer{
					{
						Source:   location.NewResource("scp://127.0.0.1:22/tmp/copy5_source/config5.json"),
						Dest:     location.NewResource("/tmp/copy5_target/config5_corruption.gz"),
						Compress: false,
					},
				},
				Udf: "CopyWithCompressionAndCorruption",
			},
			map[string]string{
				"mem:///tmp/copy5_target/config5_corruption.gz": "\x1f\x8b\b\x00\x00\tn\x88\x04\xff\x00\v\x00\xf4\xffcompressed*\x00\x00\x00\xff\xff\x00\x00\x00\xff\xff\x01\x00\x00\xff\xffsM!\xf3\v\x00\x00\x00",
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
	memStorage.Upload("mem:///test/Remove/storage/config1.json", strings.NewReader("abc"))

	object, _ := memStorage.StorageObject("mem:///test/Remove/storage/config1.json")
	assert.NotNil(t, object)

	serviceResponse := storageService.Run(context, &storage.RemoveRequest{
		Assets: []*location.Resource{
			location.NewResource("mem:///test/Remove/storage/config1.json"),
			location.NewResource("mem:///dummy"),
		},
	})
	if assert.Equal(t, serviceResponse.Error, "") {
		response, ok := serviceResponse.Response.(*storage.RemoveResponse)
		assert.EqualValues(t, 1, len(response.Removed))
		if ok {
			assert.EqualValues(t, response.Removed[0], "mem:///test/Remove/storage/config1.json")
		}
	}
	object, _ = memStorage.StorageObject("mem:///test/Remove/storage/config1.json")
	assert.Nil(t, object)

}

func TestTransferService_Download(t *testing.T) {
	var manager = endly.New()
	storageService, err := manager.Service(storage.ServiceID)
	assert.Nil(t, err)
	context := manager.NewContext(toolbox.NewContext())
	memStorage := storage.UseMemoryService(context)
	memStorage.Upload("mem:///test/Download/storage/config1.json", strings.NewReader("abc"))

	serviceResponse := storageService.Run(context, &storage.DownloadRequest{
		DestKey: "key1",
		Source:  location.NewResource("mem:///test/Download/storage/config1.json"),
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
		Dest:      location.NewResource("mem:///test/storage/Upload/config1.json"),
	})

	if assert.Equal(t, serviceResponse.Error, "") {
		response, ok := serviceResponse.Response.(*storage.UploadResponse)
		if ok {
			assert.EqualValues(t, 3, response.Size)
			assert.EqualValues(t, "mem:///test/storage/Upload/config1.json", response.URL)

			object, err := memStorage.StorageObject("mem:///test/storage/Upload/config1.json")
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
		Dest:      location.NewResource("mem:///test/storage/Upload/config1.json"),
	})
	assert.Equal(t, "sourcekey key10 value was empty at storage.Upload", serviceResponse.Error)
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
					Source: location.NewResource("abc"),
				},
			},
		}
		assert.NotNil(t, request.Validate())
	}
	{
		request := storage.CopyRequest{
			Transfers: []*storage.Transfer{
				{
					Dest: location.NewResource("abc"),
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
					Source: location.NewResource("abc"),
				},
			},
		}
		assert.NotNil(t, request.Validate())
	}
	{
		request := storage.CopyRequest{
			Transfers: []*storage.Transfer{
				{
					Dest:   location.NewResource("abc"),
					Source: location.NewResource("xyz"),
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
			Source:  location.NewResource("abc"),
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
			Dest: location.NewResource("abc"),
		}
		assert.NotNil(t, request.Validate())
	}
	{
		request := storage.UploadRequest{
			Dest:      location.NewResource("abc"),
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
			Assets: []*location.Resource{

				location.NewResource("a"),
			},
		}
		assert.Nil(t, request.Validate())
	}

}
*/
