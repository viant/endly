package endly_test

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox/storage"
	"github.com/viant/toolbox/url"
	"strings"
	"testing"
)

func TestBuildService_Build(t *testing.T) {

	var credentialFile, err = GetDummyCredential()
	assert.Nil(t, err)
	var manager = endly.NewManager()
	var useCases = []struct {
		baseDir     string
		DataURL     string
		DataPayload []byte
		target      *url.Resource
		request     *endly.BuildRequest
		expected    string
	}{
		{
			"test/build/mvn/darwin",
			"",
			[]byte{},
			url.NewResource("scp://127.0.0.1:22/tmp/project1", credentialFile),
			&endly.BuildRequest{
				BuildSpec: &endly.BuildSpec{
					Name:       "maven",
					Goal:       "build",
					BuildGoal:  "package",
					Args:       "-Dmvn.test.skip",
					Sdk:        "jdk",
					SdkVersion: "1.7",
				},
			},
			"BUILD SUCCESS",
		},

		{
			"test/build/go/linux",
			"https://redirector.gvt1.com/edgedl/go/go1.8.5.linux-amd64.tar.gz",
			[]byte("abc"),
			url.NewResource("scp://127.0.0.1:22/tmp/app", credentialFile),
			&endly.BuildRequest{
				BuildSpec: &endly.BuildSpec{
					Name:       "go",
					Goal:       "build",
					BuildGoal:  "build",
					Args:       "-o app",
					Sdk:        "go",
					SdkVersion: "1.8",
				},
			},
			"",
		},
	}

	for _, useCase := range useCases {
		execService, err := GetReplayService(useCase.baseDir)
		if assert.Nil(t, err) {
			context, err := OpenTestContext(manager, useCase.target, execService)
			var state = context.State()

			if useCase.DataURL != "" {
				storageService := storage.NewMemoryService()
				state.Put(endly.UseMemoryService, true)
				err = storageService.Upload(useCase.DataURL, bytes.NewReader(useCase.DataPayload))
				assert.Nil(t, err)
			}

			service, err := context.Service(endly.BuildServiceID)
			assert.Nil(t, err)

			defer context.Close()
			if assert.Nil(t, err) {
				serviceResponse := service.Run(context, &endly.BuildRequest{
					Target:    useCase.target,
					BuildSpec: useCase.request.BuildSpec,
				})

				var baseCase = useCase.baseDir + " " + useCase.request.BuildSpec.Name
				assert.Equal(t, "", serviceResponse.Error, baseCase)
				response, ok := serviceResponse.Response.(*endly.BuildResponse)
				if !ok {
					assert.Fail(t, fmt.Sprintf("process serviceResponse was empty %v %T", baseCase, serviceResponse.Response))
					continue
				}


				var actual = response.CommandInfo.Stdout()
				assert.True(t, strings.Contains(actual, useCase.expected), "name "+baseCase)
			}
		}
	}
}

func Test_BuildMeta_Validate(t *testing.T) {
	{
		meta := &endly.BuildMeta{}
		assert.NotNil(t, meta.Validate())
	}
	{
		meta := &endly.BuildMeta{
			Name: "abc",
		}
		assert.NotNil(t, meta.Validate())
	}
	{
		meta := &endly.BuildMeta{
			Goals: []*endly.BuildGoal{
				{
					Name: "abc",
				},
			},
		}
		assert.NotNil(t, meta.Validate())
	}


	{
		meta := &endly.BuildLoadMetaRequest{}
		assert.NotNil(t, meta.Validate())
	}

	{
		meta := &endly.BuildLoadMetaRequest{
			Source:url.NewResource("abc"),
		}
		assert.Nil(t, meta.Validate())
	}

}


func Test_BuildLoad_Validate(t *testing.T) {

	{
		request := &endly.BuildRequest{}
		assert.NotNil(t, request.Validate())
	}

	{
		request := &endly.BuildRequest{
			BuildSpec:&endly.BuildSpec{

			},
		}
		assert.NotNil(t, request.Validate())
	}

	{
		request := &endly.BuildRequest{
			BuildSpec:&endly.BuildSpec{
				Name:"abc",
			},
		}
		assert.NotNil(t, request.Validate())
	}

	{
		request := &endly.BuildRequest{
			BuildSpec:&endly.BuildSpec{
				Goal:"abc",
			},
		}
		assert.NotNil(t, request.Validate())
	}

	{
		request := &endly.BuildRequest{
			BuildSpec:&endly.BuildSpec{
				Name:"a",
				Goal:"abc",
			},
		}
		assert.Nil(t, request.Validate())
	}


}