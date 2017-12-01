package endly_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"path"
	"testing"
)



func TestService_RunStatusRequest(t *testing.T) {
	fileName, _, _ := toolbox.CallerInfo(2)
	parent, _ := path.Split(fileName)
	testProject := fmt.Sprintf("ssh://127.0.0.1/Projects/universe/backend/universe-pixel/trunk", parent)

	manager := endly.NewManager()
	service, err := manager.Service(endly.VersionControlServiceID)
	assert.Nil(t, err)
	assert.NotNil(t, service)

	context := manager.NewContext(toolbox.NewContext())
	response := service.Run(context, &endly.VcStatusRequest{
		Target: &url.Resource{
			URL:  testProject,
			Type: "svn",
		},
	})
	assert.NotNil(t, response)

	assert.Equal(t, "", response.Error)
	info, ok := response.Response.(*endly.VcInfo)
	assert.True(t, ok)
	assert.NotNil(t, info)


	//assert.Equal(t, "master", info.Branch)
	//assert.Equal(t, "3d764da443b3852260666d2c527872e2629e40e2", info.Revision)
	//assert.False(t, info.IsUptoDate)
	//assert.True(t, info.HasPendingChanges())

}

//
//func TestService_RunCheckout(t *testing.T) {
//	fileName, _, _ := toolbox.CallerInfo(2)
//	parent, _ := path.Split(fileName)
//
//	testProject1 := fmt.Sprintf("%vtest/vc/project1", parent)
//	testProject2 := fmt.Sprintf("%vtest/vc/project2", parent)
//	command := exec.Command("/bin/cp", "-rf", testProject1, testProject2)
//	_, err := command.CombinedOutput()
//	assert.Nil(t, err)
//
//	manager := endly.NewManager()
//	service, err := manager.Service(endly.VersionControlServiceID)
//	assert.Nil(t, err)
//
//	context := manager.NewContext(toolbox.NewContext())
//	response := service.Run(context, &endly.VcCheckoutRequest{
//		Origin: &url.Resource{
//			URL: "https://github.com/adranwit/p",
//		},
//		Target: &url.Resource{
//			URL:  "scp://" + testProject2,
//			Type: "git",
//		},
//	})
//	assert.Equal(t, "", response.Error)
//
//}
