package storage

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/assertly"
	"github.com/viant/toolbox/url"
	"testing"
)

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
  "Dest": {
    "URL": "mem://dest/dir2"
  },
  "Assets": {
    "file1.txt": null,
    "file2.txt": "renamedFile2"
  },
  "Transfers": [
    {
      "Source": {
        "URL": "mem://yaml1/dir/file1.txt"
      },
      "Dest": {
        "URL": "mem://dest/dir2/file1.txt"
      }
    },
    {
      "Source": {
        "URL": "mem://yaml1/dir/file2.txt"
      },
      "Dest": {
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
  "Dest": {
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
      "Dest": {
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
      "Dest": {
        "URL": "file101.txt"
      }
    },
    {
      "Expand": true,
      "Source": {
        "URL": "file2.txt"
      },
      "Dest": {
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
  "Dest": {
    "URL": "mem://dest/dir2"
  },
  "Transfers": [
    {
      "Source": {
        "URL": "mem://yaml1/dir/file1.txt"
      },
      "Dest": {
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
    "Credentials": "mem://github.com/viant/endly/workflow/docker/build/secret/build.json"
  },
  "Dest": {
    "URL": "ssh://127.0.0.1/",
    "Credentials": "/Users/awitas/.secret/localhost.json"
  },
  "Assets": {
    "/echo": "/tmp/echo/"
  },
  "Transfers": [
    {
      "Source": {
        "URL": "scp://127.0.0.1:7722/echo",
        "Credentials": "mem://github.com/viant/endly/workflow/docker/build/secret/build.json"
      },
      "Dest": {
        "URL": "ssh://127.0.0.1/tmp/echo/",
        "Credentials": "/Users/awitas/.secret/localhost.json"
      }
    }
  ]
}
`,
		},
	}

	for _, useCase := range useCases {
		request, err := NewCopyRequestFromuRL(useCase.URL)
		if useCase.HasError {
			assert.NotNil(t, err, useCase.Description)
			continue
		}
		if !assert.Nil(t, err, useCase.Description) {
			continue
		}

		assert.Nil(t, request.Init(), useCase.Description)
		assert.Nil(t, request.Validate(), useCase.Description)
		assertly.AssertValues(t, useCase.ExpectedRequest, request, useCase.Description)
	}

}

func TestCopyRequest_Validate(t *testing.T) {

	{
		var request = NewCopyRequest(nil)
		assert.NotNil(t, request.Validate())
	}
	{
		var request = NewCopyRequest(nil, NewTransfer(nil, url.NewResource("abc"), false, false, nil))
		assert.NotNil(t, request.Validate())
	}
	{
		var request = NewCopyRequest(nil, NewTransfer(url.NewResource("abc"), nil, false, false, nil))
		request.Init()
		assert.NotNil(t, request.Validate())
	}
	{
		var request = NewCopyRequest(nil, NewTransfer(url.NewResource("abc"), url.NewResource("abc"), false, false, nil))
		assert.NotNil(t, request.Validate())
	}
	{
		var request = NewCopyRequest(nil, NewTransfer(url.NewResource("abc"), url.NewResource("abc"), false, false, nil))
		request.Init()
		assert.Nil(t, request.Validate())
	}
}
