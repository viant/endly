package storage

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/model/location"
	"strings"
	"testing"
)

func TestService_Remove(t *testing.T) {

	var useCases = []struct {
		description string
		request     *RemoveRequest
		expectError bool
	}{

		{
			description: "single asset removal",
			request: &RemoveRequest{
				Assets: []*location.Resource{
					location.NewResource("mem://127.0.0.1/test/storageRemove/case001/f1"),
				},
			},
		},
		{
			description: "multi asset removal",
			request: &RemoveRequest{
				Assets: []*location.Resource{
					location.NewResource("mem://127.0.0.1/test/storageRemove/case002/f1"),
					location.NewResource("mem://127.0.0.1/test/storageRemove/case002/f2"),
				},
			},
		},
		{
			description: "empty asset error",
			request:     &RemoveRequest{},
			expectError: true,
		},
	}

	for _, useCase := range useCases {
		if !useCase.expectError && len(useCase.request.Assets) > 0 {
			for i := range useCase.request.Assets {
				_ = fs.Upload(context.Background(), useCase.request.Assets[i].URL, 0644, strings.NewReader("test"))
			}
		}
		response := &RemoveResponse{}
		err := endly.Run(nil, useCase.request, response)
		if useCase.expectError {
			assert.NotNil(t, err, useCase.description)
			continue
		}
		if !assert.Nil(t, err, useCase.description) {
			continue
		}
		assert.EqualValues(t, len(useCase.request.Assets), len(response.Removed), useCase.description)

	}

}
