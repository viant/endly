package endly_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox/url"
	"strings"
	"testing"
)

func TestBuildService_Build(t *testing.T) {

	var credentialFile, err = GetDummyCredential()
	assert.Nil(t, err)
	var target = url.NewResource("scp://127.0.0.1:22/", credentialFile) //
	var manager = endly.NewManager()
	var useCases = []struct {
		baseDir  string
		target   *url.Resource
		request  *endly.BuildRequest
		expected string
	}{
		{
			"test/build/mvn/darwin",
			target,
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
	}

	for _, useCase := range useCases {
		execService, err := GetReplayService(useCase.baseDir)
		if assert.Nil(t, err) {
			context, err := OpenTestContext(manager, useCase.target, execService)
			service, err := context.Service(endly.BuildServiceID)
			assert.Nil(t, err)

			defer context.Close()
			if assert.Nil(t, err) {
				var target = useCase.target
				serviceResponse := service.Run(context, &endly.BuildRequest{
					Target:    target,
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

//
//func TestBuildService_Run(t *testing.T) {
//
//	var credentialFile = path.Join(os.Getenv("HOME"), ".secret/scp.json")
//	var target = url.NewResource("scp://127.0.0.1:22/", credentialFile) //
//	manager := endly.NewManager()
//
//	context, err := OpenTestRecorderContext(manager, target, "test/build/svn/darwin")
//	if ! assert.Nil(t, err) || ! assert.NotNil(t, context) {
//		return
//	}
//	defer context.Close()
//
//	buildService, err := manager.Service(endly.BuildServiceID)
//	assert.Nil(t, err)
//
//	response := buildService.Run(context, &endly.BuildRequest{
//		BuildSpec: &endly.BuildSpec{
//			Name:       "maven",
//			Goal:       "build",
//			BuildGoal:  "package",
//			Args:       "-Dmvn.test.skip",
//			Sdk:        "jdk",
//			SdkVersion: "1.7",
//		},
//		Target: url.NewResource("test/build/project1"),
//	})
//	assert.Equal(t, "ok", response.Status)
//	assert.Equal(t, "", response.Error)
//
//}
