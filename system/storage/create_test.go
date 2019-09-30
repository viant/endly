package storage

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/afs/option"
	"github.com/viant/endly"
	"github.com/viant/toolbox/url"
	"io/ioutil"
	"testing"
)

func TestService_Create(t *testing.T) {

	resource := url.NewResource("mem://localhost/data/storage/create/case004/data.txt")
	resource.CustomKey = &option.AES256Key{
		Key: []byte("invalid_key"),
	}

	var useCases = []struct {
		description string
		request     *CreateRequest
		expectError bool
		data        string
	}{
		{
			description: "crate basic asset with content",
			data:        "this is test 1",
			request: &CreateRequest{
				Dest:      url.NewResource("mem://localhost/data/storage/create/case001/data.txt"),
				SourceKey: "key1",
			},
		},

		{
			description: "create directory",
			request: &CreateRequest{
				Dest:  url.NewResource("mem://localhost/data/storage/create/case002/folder1"),
				IsDir: true,
			},
		},

		{
			description: "error - dest is empty",
			request:     &CreateRequest{},
			expectError: true,
		},
		{
			description: "error invalid custom key",
			data:        "this is test 1",
			request: &CreateRequest{
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
		response := &CreateResponse{}
		err := endly.Run(ctx, useCase.request, response)
		if useCase.expectError {
			assert.NotNil(t, err, useCase.description)
			continue
		}
		if !assert.Nil(t, err, useCase.description) {
			continue
		}

		ok, err := fs.Exists(ctx.Background(), useCase.request.Dest.URL)
		if !assert.Nil(t, err, useCase.description) {
			continue
		}
		assert.True(t, ok, useCase.description)
		if useCase.request.IsDir {
			continue
		}

		reader, err := fs.DownloadWithURL(ctx.Background(), useCase.request.Dest.URL)
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
