package storage

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/afs/asset"
	"github.com/viant/afs/matcher"
	"github.com/viant/afs/mem"
	"github.com/viant/assertly"
	"github.com/viant/endly"
	"github.com/viant/endly/model/location"
	"github.com/viant/endly/service/system/storage/copy"
	"github.com/viant/toolbox"
	"testing"
)

func TestService_List(t *testing.T) {
	var useCases = []struct {
		description string
		baseURL     string
		request     *ListRequest
		prepare     []*asset.Resource
		expect      interface{}
		expectError bool
	}{
		{
			description: "single asset list",
			baseURL:     "mem://localhost/data/storage/list/case001",
			prepare: []*asset.Resource{
				asset.NewFile("f1", []byte("test1"), 0644),
			},
			request: &ListRequest{
				Source: location.NewResource("mem://localhost/data/storage/list/case001/f1"),
			},
			expect: `{
	"assets": [
		{
			"Dir": false,
			"Mode": 420,
			"Name": "mem://localhost/data/storage/list/case001/f1"
		}
	]
}`,
		},
		{
			description: "folder list",
			baseURL:     "mem://localhost/data/storage/list/case002",
			prepare: []*asset.Resource{
				asset.NewFile("f2", []byte("test1"), 0644),
			},
			request: &ListRequest{
				Source: location.NewResource("mem://localhost/data/storage/list/case002"),
			},
			expect: `{
	"assets": [
		{
			"Dir": false,
			"Name": "mem://localhost/data/storage/list/case002/f2"
		}
	]
}`,
		},

		{
			description: "folder list with matcher",
			baseURL:     "mem://localhost/data/storage/list/case003",
			prepare: []*asset.Resource{
				asset.NewFile("f1", []byte("test1"), 0644),
				asset.NewFile("f2", []byte("test1"), 0644),
				asset.NewFile("f3.txt", []byte("test1"), 0644),
			},
			request: &ListRequest{
				Source: location.NewResource("mem://localhost/data/storage/list/case003"),
				Match: &copy.Matcher{
					Basic: &matcher.Basic{Suffix: ".txt"},
				},
			},
			expect: `{
	"assets": [
		{
			"Dir": false,
			"Name": "mem://localhost/data/storage/list/case003/f3.txt"
		}
	]
}`,
		},

		{
			description: "request error",
			request:     &ListRequest{},
			expectError: true,
		},
		{
			description: "invalid path error",
			request: &ListRequest{
				Source: location.NewResource("mem://localhost/data/storage/list/case005"),
			},
			expectError: true,
		},

		{
			description: "list with content",
			baseURL:     "mem://localhost/data/storage/list/case006",
			prepare: []*asset.Resource{
				asset.NewFile("f10", []byte("test1"), 0644),
			},
			request: &ListRequest{
				Source:  location.NewResource("mem://localhost/data/storage/list/case006"),
				Content: true,
			},
			expect: `{
	"assets": [
		{
			"Dir": false,
			"Name": "mem://localhost/data/storage/list/case006/f10",	
			"TableData": "test1"
		}
	]
}`,
		},
	}

	mgr := mem.Singleton()
	for _, useCase := range useCases {
		if len(useCase.prepare) > 0 {
			err := asset.Create(mgr, useCase.baseURL, useCase.prepare)
			assert.Nil(t, err, useCase.description)
		}
		response := &ListResponse{}
		err := endly.Run(nil, useCase.request, response)
		if useCase.expectError {
			assert.NotNil(t, err, useCase.description)
			continue
		}
		if !assertly.AssertValues(t, useCase.expect, response) {
			_ = toolbox.DumpIndent(response, true)
		}
	}

}
