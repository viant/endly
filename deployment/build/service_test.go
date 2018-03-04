package build_test

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	tstorage "github.com/viant/toolbox/storage"
	"github.com/viant/toolbox/url"
	"strings"
	"testing"
	"github.com/viant/endly/util"
	"github.com/viant/endly/deployment/build"
	"github.com/viant/endly/system/exec"
	"github.com/viant/endly/system/storage"
)

func TestBuildService_Build(t *testing.T) {

	var credentialFile, err = util.GetDummyCredential()
	assert.Nil(t, err)
	var manager = endly.NewManager()
	var useCases = []struct {
		baseDir     string
		DataURL     string
		DataPayload []byte
		target      *url.Resource
		request     *build.Request
		expected    string
	}{
		{
			"test/mvn/darwin",
			"",
			[]byte{},
			url.NewResource("scp://127.0.0.1:22/tmp/project1", credentialFile),
			&build.Request{
				BuildSpec: &build.Spec{
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
			"test/go/linux",
			"https://redirector.gvt1.com/edgedl/go/go1.8.5.linux-amd64.tar.gz",
			[]byte("abc"),
			url.NewResource("scp://127.0.0.1:22/tmp/app", credentialFile),
			&build.Request{
				BuildSpec: &build.Spec{
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
		execService, err := exec.GetReplayService(useCase.baseDir)
		if assert.Nil(t, err) {
			context, err := exec.OpenTestContext(manager, useCase.target, execService)
			var state = context.State()

			if useCase.DataURL != "" {
				storageService := tstorage.NewMemoryService()
				state.Put(storage.UseMemoryService, true)
				err = storageService.Upload(useCase.DataURL, bytes.NewReader(useCase.DataPayload))
				assert.Nil(t, err)
			}

			service, err := context.Service(build.ServiceID)
			assert.Nil(t, err)

			defer context.Close()
			if assert.Nil(t, err) {
				serviceResponse := service.Run(context, &build.Request{
					Target:    useCase.target,
					BuildSpec: useCase.request.BuildSpec,
				})

				var baseCase = useCase.baseDir + " " + useCase.request.BuildSpec.Name
				assert.Equal(t, "", serviceResponse.Error, baseCase)
				response, ok := serviceResponse.Response.(*build.Response)
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
		meta := &build.Meta{}
		assert.NotNil(t, meta.Validate())
	}
	{
		meta := &build.Meta{
			Name: "abc",
		}
		assert.NotNil(t, meta.Validate())
	}
	{
		meta := &build.Meta{
			Goals: []*build.Goal{
				{
					Name: "abc",
				},
			},
		}
		assert.NotNil(t, meta.Validate())
	}

	{
		meta := &build.LoadMetaRequest{}
		assert.NotNil(t, meta.Validate())
	}

	{
		meta := &build.LoadMetaRequest{
			Source: url.NewResource("abc"),
		}
		assert.Nil(t, meta.Validate())
	}

}

func Test_BuildLoad_Validate(t *testing.T) {

	{
		request := &build.Request{}
		assert.NotNil(t, request.Validate())
	}

	{
		request := &build.Request{
			BuildSpec: &build.Spec{},
		}
		assert.NotNil(t, request.Validate())
	}

	{
		request := &build.Request{
			BuildSpec: &build.Spec{
				Name: "abc",
			},
		}
		assert.NotNil(t, request.Validate())
	}

	{
		request := &build.Request{
			BuildSpec: &build.Spec{
				Goal: "abc",
			},
		}
		assert.NotNil(t, request.Validate())
	}

	{
		request := &build.Request{
			BuildSpec: &build.Spec{
				Name: "a",
				Goal: "abc",
			},
		}
		assert.Nil(t, request.Validate())
	}

}
