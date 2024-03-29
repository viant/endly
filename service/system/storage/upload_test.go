package storage

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/afs/option"
	"github.com/viant/endly"
	"github.com/viant/endly/model/location"
	"io/ioutil"
	"testing"
)

func TestService_Upload(t *testing.T) {

	resource := location.NewResource("mem://localhost/data/storage/upload/case004/data.txt")
	resource.CustomKey = &option.AES256Key{
		Key: []byte("invalid_key"),
	}

	var useCases = []struct {
		description string
		request     *UploadRequest
		expectError bool
		data        string
	}{
		{
			description: "basic upload",
			data:        "this is test 1",
			request: &UploadRequest{
				Dest:      location.NewResource("mem://localhost/data/storage/upload/case001/data.txt"),
				SourceKey: "key1",
			},
		},
		{
			description: "error - key is empty",
			data:        "this is test 1",
			request: &UploadRequest{
				Dest: location.NewResource("mem://localhost/data/storage/upload/case002/data.txt"),
			},
			expectError: true,
		},
		{
			description: "error invalid custom key",
			data:        "this is test 1",
			request: &UploadRequest{
				Dest:      resource,
				SourceKey: "key2",
			},
			expectError: true,
		},
	}

	manager := endly.New()
	for _, useCase := range useCases {
		ctx := manager.NewContext(nil)
		if useCase.request.SourceKey != "" {
			state := ctx.State()
			state.Put(useCase.request.SourceKey, useCase.data)
		}
		response := &UploadResponse{}
		err := endly.Run(ctx, useCase.request, response)
		if useCase.expectError {
			assert.NotNil(t, err, useCase.description)
			continue
		}
		if !assert.Nil(t, err, useCase.description) {
			continue
		}
		reader, err := fs.OpenURL(ctx.Background(), useCase.request.Dest.URL)
		if !assert.Nil(t, err, useCase.description) {
			continue
		}
		actual, err := ioutil.ReadAll(reader)
		if !assert.Nil(t, err, useCase.description) {
			continue
		}
		assert.EqualValues(t, useCase.data, string(actual), useCase.data)
	}

}
