package storage

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox/url"
	"strings"
	"testing"
)

func TestService_Exists(t *testing.T) {

	var useCases = []struct {
		description string
		request     *ExistsRequest
		populate    []*location.Resource
		expectError bool
		expect      map[string]bool
	}{
		{
			description: "single asset exists",
			populate: []*location.Resource{
				url.NewResource("mem://127.0.0.1/test/storage/exists/case001/f1"),
			},
			expect: map[string]bool{
				url.NewResource("mem://127.0.0.1/test/storage/exists/case001/f1").URL: true,
			},
			request: &ExistsRequest{
				Assets: []*location.Resource{
					url.NewResource("mem://127.0.0.1/test/storage/exists/case001/f1"),
				},
			},
		},
		{
			description: "multi asset exists",
			populate: []*location.Resource{
				url.NewResource("mem://127.0.0.1/test/storage/exists/case002/f1"),
			},
			expect: map[string]bool{
				url.NewResource("mem://127.0.0.1/test/storage/exists/case002/f1").URL: true,
				url.NewResource("mem://127.0.0.1/test/storage/exists/case002/f2").URL: false,
			},
			request: &ExistsRequest{
				Assets: []*location.Resource{
					url.NewResource("mem://127.0.0.1/test/storage/exists/case002/f1"),
					url.NewResource("mem://127.0.0.1/test/storage/exists/case002/f2"),
				},
			},
		},

		{
			description: "invalid request error",
			request:     &ExistsRequest{},
			expectError: true,
		},
		{
			description: "invalid asset URL error",
			request: &ExistsRequest{
				Assets: []*location.Resource{
					url.NewResource("mem://127.0.0.1/test/storage/exists/case003/f1"),
					url.NewResource("err://127.0.0.1/test/storage/exists/case003/f2"),
				},
			},
			expectError: true,
		},
		{
			description: "multi asset exists with assert",
			populate: []*location.Resource{
				url.NewResource("mem://127.0.0.1/test/storage/exists/case004/f1"),
			},
			expect: map[string]bool{
				url.NewResource("mem://127.0.0.1/test/storage/exists/case004/f1").URL: true,
				url.NewResource("mem://127.0.0.1/test/storage/exists/case004/f2").URL: false,
			},
			request: &ExistsRequest{
				Assets: []*location.Resource{
					url.NewResource("mem://127.0.0.1/test/storage/exists/case004/f1"),
					url.NewResource("mem://127.0.0.1/test/storage/exists/case004/f2"),
				},
				Expect: map[string]bool{
					url.NewResource("mem://127.0.0.1/test/storage/exists/case004/f1").URL: true,
					url.NewResource("mem://127.0.0.1/test/storage/exists/case004/f2").URL: false,
				},
			},
		},
	}

	for _, useCase := range useCases {
		if len(useCase.populate) > 0 {
			for _, resource := range useCase.populate {
				_ = fs.Upload(context.Background(), resource.URL, 0644, strings.NewReader("test"))
			}
		}
		response := &ExistsResponse{}
		err := endly.Run(nil, useCase.request, response)
		if useCase.expectError {
			assert.NotNil(t, err, useCase.description)
			continue
		}

		if !assert.Nil(t, err, useCase.description) {
			continue
		}

		assert.EqualValues(t, useCase.expect, response.Exists, useCase.description)
	}
}
