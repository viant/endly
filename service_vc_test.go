package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox/url"
	"testing"
	"fmt"
)


func TestVc_Status(t *testing.T) {
	credentialFile, err := GetDummyCredential()
	assert.Nil(t, err)
	var target = url.NewResource("ssh://127.0.0.1/Projects/project1/trunk", credentialFile) //
	target.Type = "svn"
	var manager = endly.NewManager()
	var useCases = []struct {
		baseDir  string
		Request  *endly.VcStatusRequest
		Expected *endly.VcInfo
		Error    string
	}{
		{
			"test/vc/svn/status/darwin",
			&endly.VcStatusRequest{
				Target:     target,
			},
			&endly.VcInfo{
				Untracked:[]string{".idea"},
				New:[]string{"test.dep"},
				Modified:[]string{"pom.xml", "src/main/java/com/viant/Dummy.java"},
				Deleted:[]string{"src/test/java/com/viant/DummyTest.java"},
				IsVersionControlManaged:true,
				Origin:"http://svn.viant.com/svn/projects/project1/trunk",
				Branch:"trunk",
			},

			"",
		},
	}

	for _, useCase := range useCases {
		var target = useCase.Request.Target
		execService, err := GetReplayService(useCase.baseDir)
		if assert.Nil(t, err) {
			context, err := OpenTestContext(manager, target, execService)
			service, err := context.Service(endly.VersionControlServiceID)
			assert.Nil(t, err)

			defer context.Close()
			if assert.Nil(t, err) {
				serviceResponse := service.Run(context, useCase.Request)

				var baseCase = useCase.baseDir + " "
				assert.Equal(t, useCase.Error, serviceResponse.Error, baseCase)

				actual, ok := serviceResponse.Response.(*endly.VcInfo)
				if !ok {
					assert.Fail(t, fmt.Sprintf("process serviceResponse was empty %v %T", baseCase, serviceResponse.Response))
					continue
				}

				if actual == nil {
					continue
				}
				var expected = useCase.Expected
				assert.Equal(t, expected.Untracked, actual.Untracked, "Untracked "+baseCase)
				assert.Equal(t, expected.New, actual.New, "New "+baseCase)
				assert.Equal(t, expected.Modified, actual.Modified, "Modified "+baseCase)
				assert.Equal(t, expected.Deleted, actual.Deleted, "Deleted "+baseCase)
				assert.Equal(t, expected.IsVersionControlManaged, actual.IsVersionControlManaged, "IsVersionControlManaged "+baseCase)
				assert.Equal(t, expected.IsUptoDate, actual.IsUptoDate, "IsUptoDate "+baseCase)
				assert.Equal(t, expected.Origin, actual.Origin, "Origin "+baseCase)
				assert.Equal(t, expected.Branch, actual.Branch, "Branch "+baseCase)

			}
		}
	}
}


func TestVc_Checkout(t *testing.T) {
	credentialFile, err := GetDummyCredential()
	assert.Nil(t, err)
	var target = url.NewResource("ssh://127.0.0.1/Projects/project1/trunk", credentialFile) //
	target.Type = "svn"
	var manager = endly.NewManager()
	var useCases = []struct {
		baseDir  string
		Request  *endly.VcCheckoutRequest
		Expected *endly.VcCheckoutResponse
		Error    string
	}{
		{
			"test/vc/svn/checkout/error/darwin",
			&endly.VcCheckoutRequest{
				Target:     url.NewResource("scp://127.0.0.1:22/tmp/project2/trunk", credentialFile),
				Origin:url.NewResource("http://svn.viant.com/svn/projects/project1/trunk", credentialFile),

			},
			&endly.VcCheckoutResponse{

				//Untracked:[]string{".idea"},
				//New:[]string{"test.dep"},
				//Modified:[]string{"pom.xml", "src/main/java/com/viant/Dummy.java"},
				//Deleted:[]string{"src/test/java/com/viant/DummyTest.java"},
				//IsVersionControlManaged:true,
				//Origin:"http://svn.viant.com/svn/projects/project1/trunk",
				//Branch:"trunk",
			},

			"failed to checkout version: http://svn.viant.com/svn/projects/project1/trunk -> scp://127.0.0.1:22/tmp/project2/trunk, failed to authenticate username: awitas with dummy9.json secret",
		},
		{
			"test/vc/svn/checkout/new/darwin",
			&endly.VcCheckoutRequest{
				Target:     url.NewResource("scp://127.0.0.1:22/tmp/project1/trunk", credentialFile),
				Origin:url.NewResource("http://svn.viant.com/svn/projects/project1/trunk", credentialFile),

			},
			&endly.VcCheckoutResponse{
				Checkouts:map[string]*endly.VcInfo{
					"http://svn.viant.com/svn/projects/project1/trunk":&endly.VcInfo{
						Origin:"http://svn.viant.com/svn/projects/project1/trunk",
						IsVersionControlManaged:true,
						IsUptoDate:true,
						Modified:[]string{},
						Branch:"trunk",
					},
				},
			},
			"",
		},
		{
			"test/vc/svn/checkout/existing/darwin",
			&endly.VcCheckoutRequest{
				Target:     url.NewResource("scp://127.0.0.1:22/tmp/project1/trunk", credentialFile),
				Origin:url.NewResource("http://svn.viant.com/svn/projects/project1/trunk", credentialFile),

			},
			&endly.VcCheckoutResponse{
				Checkouts:map[string]*endly.VcInfo{
					"http://svn.viant.com/svn/projects/project1/trunk":&endly.VcInfo{
						Origin:"http://svn.viant.com/svn/projects/project1/trunk",
						IsVersionControlManaged:true,
						IsUptoDate:false,
						Branch:"trunk",
						Modified:[]string{"pom.xml"},
					},
				},
			},
			"",
		},
		{
			"test/vc/svn/checkout/modules/darwin",
			&endly.VcCheckoutRequest{
				Target:     url.NewResource("scp://127.0.0.1:22/tmp/project3/", credentialFile),
				Origin:url.NewResource("http://svn.viant.com/svn/projects/", credentialFile),
				Modules:[]string{"project1/trunk", "project2/trunk"},
			},
			&endly.VcCheckoutResponse{
				Checkouts:map[string]*endly.VcInfo{
					"http://svn.viant.com/svn/projects/project1/trunk":&endly.VcInfo{
						Origin:"http://svn.viant.com/svn/projects/project1/trunk",
						IsVersionControlManaged:true,
						IsUptoDate:true,
						Branch:"trunk",
						Modified:[]string{},
					},
					"http://svn.viant.com/svn/projects/project2/trunk":&endly.VcInfo{
						Origin:"http://svn.viant.com/svn/projects/project2/trunk",
						IsVersionControlManaged:true,
						IsUptoDate:true,
						Branch:"trunk",
						Modified:[]string{},
					},

				},
			},
			"",
		},
	}

	for _, useCase := range useCases {
		var target = useCase.Request.Target
		execService, err := GetReplayService(useCase.baseDir)
		if assert.Nil(t, err) {
			context, err := OpenTestContext(manager, target, execService)
			service, err := context.Service(endly.VersionControlServiceID)
			assert.Nil(t, err)

			defer context.Close()
			if assert.Nil(t, err) {
				serviceResponse := service.Run(context, useCase.Request)

				var baseCase = useCase.baseDir + " "
				assert.Equal(t, useCase.Error, serviceResponse.Error, baseCase)

				response, ok := serviceResponse.Response.(*endly.VcCheckoutResponse)
				if !ok {
					assert.Fail(t, fmt.Sprintf("process serviceResponse was empty %v %T", baseCase, serviceResponse.Response))
					continue
				}

				if response == nil {
					continue
				}
				assert.EqualValues(t, useCase.Error, serviceResponse.Error)

				if useCase.Error != "" {
					continue
				}
				for key, actual := range response.Checkouts {

					expected, ok := useCase.Expected.Checkouts[key]
					if ! assert.True(t, ok, "missing origin: "+ key) {
						continue
					}
					assert.Equal(t, expected.IsVersionControlManaged, actual.IsVersionControlManaged, "IsVersionControlManaged "+baseCase)
					assert.Equal(t, expected.IsUptoDate, actual.IsUptoDate, "IsUptoDate "+baseCase)
					assert.Equal(t, expected.Origin, actual.Origin, "Origin "+baseCase)
					assert.Equal(t, expected.Branch, actual.Branch, "Branch "+baseCase)
					assert.Equal(t, expected.Modified, actual.Modified, "Modified "+baseCase)
				}




			}
		}
	}
}



//
//func TestService_Run2StatusRequest(t *testing.T) {
//
//
//	var credentialFile = path.Join(os.Getenv("HOME"), ".secret/scp.json")
//	var svnCredentialFile = path.Join(os.Getenv("HOME"), ".secret/vsvn.json")
//
//
//	//var target = url.NewResource("scp://35.197.115.53:22/", credentialFile) //
//	var target = url.NewResource("scp://127.0.0.1:22/tmp/project3/", credentialFile) //
//	manager := endly.NewManager()
//
//	context, err := OpenTestRecorderContext(manager, target, "test/vc/svn/checkout/modules/darwin")
//
//	defer context.Close()
//
//	service, err := manager.Service(endly.VersionControlServiceID)
//	assert.Nil(t, err)
//	assert.NotNil(t, service)
//	var origin = url.NewResource("http://svn.viant.com/svn/projects/", svnCredentialFile)
//	origin.Type = "svn"
//
//	response := service.Run(context, &endly.VcCheckoutRequest{
//		Target: target,
//		Origin:origin,
//		Modules:[]string{"project1/trunk", "project2/trunk"},
//		RemoveLocalChanges:true,
//	})
//	assert.NotNil(t, response)
//	assert.Equal(t, "", response.Error)
//	info, ok := response.Response.(*endly.VcInfo)
//	assert.True(t, ok)
//	assert.NotNil(t, info)
//
//	//assert.Equal(t, "master", info.Branch)
//	//assert.Equal(t, "3d764da443b3852260666d2c527872e2629e40e2", info.Revision)
//	//assert.False(t, info.IsUptoDate)
//	//assert.True(t, info.HasPendingChanges())
//
//}


//
//
//
////
////func TestService_RunCheckout(t *testing.T) {
////	fileName, _, _ := toolbox.CallerInfo(2)
////	parent, _ := path.Split(fileName)
////
////	testProject1 := fmt.Sprintf("%vtest/vc/project1", parent)
////	testProject2 := fmt.Sprintf("%vtest/vc/project2", parent)
////	command := exec.Command("/bin/cp", "-rf", testProject1, testProject2)
////	_, err := command.CombinedOutput()
////	assert.Nil(t, err)
////
////	manager := endly.NewManager()
////	service, err := manager.Service(endly.VersionControlServiceID)
////	assert.Nil(t, err)
////
////	context := manager.NewContext(toolbox.NewContext())
////	response := service.Run(context, &endly.VcCheckoutRequest{
////		Origin: &url.Resource{
////			URL: "https://github.com/adranwit/p",
////		},
////		Target: &url.Resource{
////			URL:  "scp://" + testProject2,
////			Type: "git",
////		},
////	})
////	assert.Equal(t, "", response.Error)
////
////}
