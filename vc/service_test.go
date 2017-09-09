package vc_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/vc"
	"github.com/viant/toolbox"
	"os/exec"
	"path"
	"testing"
)

func TestService_RunStatusRequest(t *testing.T) {
	fileName, _, _ := toolbox.CallerInfo(2)
	parent, _ := path.Split(fileName)
	testProject := fmt.Sprintf("ssh://%vtest/project1", parent)

	manager := endly.NewManager()
	service, err := manager.Service(vc.VersionControlServiceId)
	assert.Nil(t, err)
	assert.NotNil(t, service)

	context := manager.NewContext(toolbox.NewContext())
	response := service.Run(context, &vc.StatusRequest{
		Target: &endly.Resource{
			URL:  testProject,
			Type: "git",
		},
	})
	assert.NotNil(t, response)

	assert.Nil(t, response.Error)
	info, ok := response.Response.(*vc.InfoResponse)
	assert.True(t, ok)
	assert.Equal(t, "master", info.Branch)
	assert.Equal(t, "68a240190783eacdeb510098e9cc3b5a4b58d1d8", info.Revision)
	assert.False(t, info.IsUptoDate)
	assert.True(t, info.HasPendingChanges())

}

func TestService_RunCheckout(t *testing.T) {
	fileName, _, _ := toolbox.CallerInfo(2)
	parent, _ := path.Split(fileName)

	testProject1 := fmt.Sprintf("%vtest/project1", parent)
	testProject2 := fmt.Sprintf("%vtest/project2", parent)
	command := exec.Command("/bin/cp", "-rf", testProject1, testProject2)
	_, err := command.CombinedOutput()
	assert.Nil(t, err)

	manager := endly.NewManager()
	service, err := manager.Service(vc.VersionControlServiceId)
	assert.Nil(t, err)

	context := manager.NewContext(toolbox.NewContext())
	response := service.Run(context, &vc.CheckoutRequest{
		Origin: &endly.Resource{
			URL: "https://github.com/adranwit/p",
		},
		Target: &endly.Resource{
			URL:  "scp://" + testProject2,
			Type: "git",
		},
	})
	assert.Nil(t, response.Error)

}
