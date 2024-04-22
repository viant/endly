package storage

import (
	"fmt"
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

func TestService_Copy(t *testing.T) {

	var useCases = []struct {
		description string
		baseURL     string
		destURL     string
		request     *CopyRequest
		prepare     []*asset.Resource
		expect      []*asset.Resource
		expectError bool
	}{
		{
			description: "folder copy",
			baseURL:     "mem://localhost/data/storage/copy/case001/src",
			destURL:     "mem://localhost/data/storage/copy/case001/dst",
			prepare: []*asset.Resource{
				asset.NewFile("f1", []byte("test1"), 0644),
				asset.NewFile("f2", []byte("test2"), 0644),
			},
			request: &CopyRequest{
				Rule: &copy.Rule{
					Source: location.NewResource("mem://localhost/data/storage/copy/case001/src"),
					Dest:   location.NewResource("mem://localhost/data/storage/copy/case001/dst"),
				},
			},
		},
		{
			description: "asset copy",
			baseURL:     "mem://localhost/data/storage/copy/case002/src",
			destURL:     "mem://localhost/data/storage/copy/case002/dst",
			prepare: []*asset.Resource{
				asset.NewFile("f1", []byte("test1"), 0644),
				asset.NewFile("f2", []byte("test2"), 0644),
			},
			expect: []*asset.Resource{
				asset.NewFile("f1", []byte("test1"), 0644),
			},
			request: &CopyRequest{
				Rule: &copy.Rule{
					Source: location.NewResource("mem://localhost/data/storage/copy/case002/src/f1"),
					Dest:   location.NewResource("mem://localhost/data/storage/copy/case002/dst"),
				},
			},
		},

		{
			description: "folder copy with matcher",
			baseURL:     "mem://localhost/data/storage/copy/case003/src",
			destURL:     "mem://localhost/data/storage/copy/case003/dst",
			prepare: []*asset.Resource{
				asset.NewFile("f1", []byte("test1"), 0644),
				asset.NewFile("f2.txt", []byte("test2"), 0644),
				asset.NewFile("f3", []byte("test3"), 0644),
				asset.NewFile("f4.txt", []byte("test4"), 0644),
			},
			expect: []*asset.Resource{
				asset.NewFile("f2.txt", []byte("test2"), 0644),
				asset.NewFile("f4.txt", []byte("test4"), 0644),
			},
			request: &CopyRequest{
				Rule: &copy.Rule{
					Source: location.NewResource("mem://localhost/data/storage/copy/case003/src"),
					Dest:   location.NewResource("mem://localhost/data/storage/copy/case003/dst"),
					Matcher: &copy.Matcher{
						Basic: &matcher.Basic{Suffix: ".txt"},
					},
				},
			},
		},

		{
			description: "folder copy with modifier",
			baseURL:     "mem://localhost/data/storage/copy/case004/src",
			destURL:     "mem://localhost/data/storage/copy/case004/dst",
			prepare: []*asset.Resource{
				asset.NewFile("f1", []byte("test1"), 0644),
				asset.NewFile("f2.txt", []byte("test2"), 0644),
				asset.NewFile("f3", []byte("test3"), 0644),
				asset.NewFile("f4.txt", []byte("test4"), 0644),
			},
			expect: []*asset.Resource{
				asset.NewFile("f2.txt", []byte("test2"), 0644),
				asset.NewFile("f4.txt", []byte("replaced from"), 0644),
			},
			request: &CopyRequest{
				Rule: &copy.Rule{
					Source: location.NewResource("mem://localhost/data/storage/copy/case004/src"),
					Dest:   location.NewResource("mem://localhost/data/storage/copy/case004/dst"),
					Matcher: &copy.Matcher{
						Basic: &matcher.Basic{Suffix: ".txt"},
					},
					Substitution: copy.Substitution{
						Replace: map[string]string{
							"test4": "replaced from",
						},
					},
				},
			},
		},

		{
			description: "folder copy with conditional modifier",
			baseURL:     "mem://localhost/data/storage/copy/case005/src",
			destURL:     "mem://localhost/data/storage/copy/case005/dst",
			prepare: []*asset.Resource{
				asset.NewFile("f1", []byte("test1"), 0644),
				asset.NewFile("f2.txt", []byte("test2"), 0644),
				asset.NewFile("f3", []byte("test3"), 0644),
				asset.NewFile("f4.txt", []byte("test4"), 0644),
			},
			expect: []*asset.Resource{
				asset.NewFile("f2.txt", []byte("test2"), 0644),
				asset.NewFile("f4.txt", []byte("replaced from4"), 0644),
			},
			request: &CopyRequest{
				Rule: &copy.Rule{
					Source: location.NewResource("mem://localhost/data/storage/copy/case005/src"),
					Dest:   location.NewResource("mem://localhost/data/storage/copy/case005/dst"),
					Matcher: &copy.Matcher{
						Basic: &matcher.Basic{Suffix: ".txt"},
					},
					Substitution: copy.Substitution{
						Replace: map[string]string{
							"test": "replaced from",
						},
						ExpandIf: &copy.Matcher{
							Basic: &matcher.Basic{Suffix: "f4.txt"},
						},
					},
				},
			},
		},
	}

	mgr := mem.Singleton()
	for _, useCase := range useCases {
		if len(useCase.prepare) > 0 {
			err := asset.Create(mgr, useCase.baseURL, useCase.prepare)
			assert.Nil(t, err, useCase.description)
		}

		err := useCase.request.Init()
		if !assert.Nil(t, err, useCase.description) {
			continue
		}

		response := &CopyResponse{}
		err = endly.Run(nil, useCase.request, response)
		if useCase.expectError {
			assert.NotNil(t, err, useCase.description)
			continue
		}
		if !assert.Nil(t, err, useCase.description) {
			continue
		}

		assets, err := asset.Load(mgr, useCase.destURL)
		if !assert.Nil(t, err, useCase.description) {
			continue
		}

		if len(useCase.expect) == 0 {
			useCase.expect = useCase.prepare
		}

		assert.EqualValues(t, len(useCase.expect), len(assets), useCase.description)

		for _, expect := range useCase.expect {
			actual, ok := assets[expect.Name]
			description := useCase.description + " / " + expect.Name
			if !assert.True(t, ok, description) {
				_ = toolbox.DumpIndent(assets, true)
				continue
			}
			assert.EqualValues(t, actual.Mode, expect.Mode, description)
			assert.EqualValues(t, actual.Data, expect.Data, description)

		}

	}

}

func TestNewCopyRequestFromURL(t *testing.T) {

	var useCases = []struct {
		Description     string
		URL             string
		ExpectedRequest interface{}
		HasError        bool
	}{
		{
			Description: "yaml asset request",
			URL:         "test/copy1.yaml",
			ExpectedRequest: `{
  "Source": {
    "URL": "mem://yaml1/dir"
  },
  "Target": {
    "URL": "mem://dest/dir2"
  },
  "assets": {
    "file2.txt": "renamedFile2"
  },
  "Transfers": [
   {
      "Source": {
        "URL": "mem://yaml1/dir/file2.txt"
      },
      "Target": {
        "URL": "mem://dest/dir2/renamedFile2"
      }
    }
  ]
}`,
		},
		{
			Description: "json root source and dest only",
			URL:         "test/copy2.json",
			ExpectedRequest: `{
  "Source": {
    "URL": "mem://yaml1/dir"
  },
  "Target": {
    "URL": "mem://dest/dir2"
  },
  "Compress": true,
  "Expand": true,
  "Replace": {
    "k1": "1",
    "k2": "2"
  },
  "Transfers": [
    {
      "Source": {
        "URL": "mem://yaml1/dir"
      },
      "Target": {
        "URL": "mem://dest/dir2"
      }
    }
  ]
}`,
		},

		{
			Description: "yaml transfers request",
			URL:         "test/copy3.yaml",
			ExpectedRequest: `{
  "Transfers": [
    {
      "Expand": true,
      "Source": {
        "URL": "file1.txt"
      },
      "Target": {
        "URL": "file101.txt"
      }
    },
    {
      "Expand": true,
      "Source": {
        "URL": "file2.txt"
      },
      "Target": {
        "URL": "file201.txt"
      }
    }
  ]
}`,
		},

		{
			Description: "yaml transfers relative path update",
			URL:         "test/copy4.json",
			ExpectedRequest: `{
  "Source": {
    "URL": "mem://yaml1/dir"
  },
  "Target": {
    "URL": "mem://dest/dir2"
  },
  "Transfers": [
    {
      "Source": {
        "URL": "mem://yaml1/dir/file1.txt"
      },
      "Target": {
        "URL": "mem://dest/file1.txt"
      }
    }
  ]
}
`,
		},
		{
			Description: "yaml asset relative path update",
			URL:         "test/copy5.yaml",
			ExpectedRequest: `{
  "Source": {
    "URL": "scp://127.0.0.1:7722/echo",
    "Credentials": "mem://github.com/viant/endly/service/workflow/docker/build/secret/build.json"
  },
  "Target": {
    "URL": "ssh://127.0.0.1/",
    "Credentials": "/Users/awitas/.secret/localhost.json"
  },
  "assets": {
    "/echo": "/tmp/echo/"
  },
  "Transfers": [
    {
      "Source": {
        "URL": "scp://127.0.0.1:7722/echo",
        "Credentials": "mem://github.com/viant/endly/service/workflow/docker/build/secret/build.json"
      },
      "Target": {
        "URL": "ssh://127.0.0.1/tmp/echo/",
        "Credentials": "/Users/awitas/.secret/localhost.json"
      }
    }
  ]
}
`,
		},
	}

	for i, useCase := range useCases {
		request, err := NewCopyRequestFromURL(useCase.URL)
		if useCase.HasError {
			assert.NotNil(t, err, useCase.Description)
			continue
		}
		if !assert.Nil(t, err, useCase.Description) {
			continue
		}

		assert.Nil(t, request.Init(), useCase.Description)
		assert.Nil(t, request.Validate(), useCase.Description)
		assertly.AssertValues(t, useCase.ExpectedRequest, request, useCase.Description+fmt.Sprintf("[%d]", i))
	}

}

func TestCopyRequest_Validate(t *testing.T) {
	{
		var request = NewCopyRequest(nil)
		assert.NotNil(t, request.Validate())
	}
	{
		var request = NewCopyRequest(nil, copy.New(nil, location.NewResource("abc"), false, false, nil))
		assert.NotNil(t, request.Validate())
	}
	{
		var request = NewCopyRequest(nil, copy.New(location.NewResource("abc"), nil, false, false, nil))
		err := request.Init()
		assert.Nil(t, err)
		assert.NotNil(t, request.Validate())
	}
	{
		var request = NewCopyRequest(nil, copy.New(location.NewResource("abc"), location.NewResource("abc"), false, false, nil))
		assert.NotNil(t, request.Validate())
	}
	{
		var request = NewCopyRequest(nil, copy.New(location.NewResource("abc"), location.NewResource("abc"), false, false, nil))
		err := request.Init()
		assert.Nil(t, err)
		assert.Nil(t, request.Validate())
	}
}
