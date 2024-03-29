package storage

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/model/location"
	"strings"
	"testing"
)

func TestService_Download(t *testing.T) {

	var useCases = []struct {
		description string
		request     *DownloadRequest
		data        string
		expectState interface{}
		expectError bool
	}{
		{
			description: "basic asset download",
			data:        "this is test",
			request: &DownloadRequest{
				Source: location.NewResource("mem://127.0.0.1/test/storage/download/case001/f1"),
			},
		},
		{
			description: "basic asset download with dest key",
			data:        "this is test",
			request: &DownloadRequest{
				Source:  location.NewResource("mem://127.0.0.1/test/storage/download/case002/f1"),
				DestKey: "k2",
			},
		},
		{
			description: "basic asset download with AsNode udf",
			data:        `{"k1":123}`,
			expectState: map[string]interface{}{
				"k1": 123,
			},
			request: &DownloadRequest{
				Udf:     "AsNode",
				Source:  location.NewResource("mem://127.0.0.1/test/storage/download/case003/f1"),
				DestKey: "k2",
			},
		},
		{
			description: "udf error",
			data:        "this is test",
			expectError: true,
			request: &DownloadRequest{
				Udf:    "blahblah",
				Source: location.NewResource("mem://127.0.0.1/test/storage/download/case004/f1"),
			},
		},
		{
			description: "request error",
			data:        "this is test",
			expectError: true,
			request:     &DownloadRequest{},
		},
		{
			description: "transformation error AsNode udf",
			data:        `[1,2]`,
			expectError: true,
			request: &DownloadRequest{
				Udf:     "AsNode",
				Source:  location.NewResource("mem://127.0.0.1/test/storage/download/case006/f1"),
				DestKey: "k2",
			},
		},
		{
			description: "download asset error",
			data:        "",
			expectError: true,
			request: &DownloadRequest{
				Source: location.NewResource("mem://127.0.0.1/test/storage/download/case007/f1"),
			},
		},
		{
			description: "basic asset download with assert",
			data:        "this is test",
			request: &DownloadRequest{
				Source: location.NewResource("mem://127.0.0.1/test/storage/download/case008/f1"),
				Expect: "/test/",
			},
		},
	}
	manager := endly.New()
	for _, useCase := range useCases {

		if useCase.request.Source != nil && useCase.data != "" {
			_ = fs.Upload(context.Background(), useCase.request.Source.URL, 0644, strings.NewReader(useCase.data))
		}

		ctx := manager.NewContext(nil)
		response := &DownloadResponse{}
		err := endly.Run(ctx, useCase.request, response)
		if useCase.expectError {
			assert.NotNil(t, err, useCase.description)
			continue
		}

		if !assert.Nil(t, err, useCase.description) {
			continue
		}
		if useCase.expectState == nil {
			useCase.expectState = useCase.data
		}
		if useCase.request.Udf != "" {
			assert.EqualValues(t, useCase.expectState, response.Transformed, useCase.description)
		}
		assert.EqualValues(t, useCase.data, response.Payload, useCase.description)
		if useCase.request.DestKey != "" {
			state := ctx.State()
			destValue := state.Get(useCase.request.DestKey)
			assert.EqualValues(t, useCase.expectState, destValue, useCase.description)
		}

	}
}
