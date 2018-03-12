package sdk_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/deployment/sdk"
	_ "github.com/viant/endly/static"
	"github.com/viant/endly/system/exec"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox/url"
	"log"
	"testing"
)

func TestSdkService_Run_Jdk(t *testing.T) {

	manager := endly.New()
	var credential, err = util.GetDummyCredential()
	if err != nil {
		log.Fatal(err)
	}
	target := url.NewResource("ssh://127.0.0.1", credential)
	SSHService, err := exec.GetReplayService("test/jdk")
	if err != nil {
		log.Fatal(err)
	}
	context, err := exec.OpenTestContext(manager, target, SSHService)
	if err != nil {
		log.Fatal(err)
	}

	sdkResponse := &sdk.SetResponse{}
	err = endly.Run(context, sdk.NewSetRequest(target, "jdk", "1.7", nil), sdkResponse)
	assert.Nil(t, err)
	assert.EqualValues(t, "/Library/Java/JavaVirtualMachines/jdk1.7.0_80.jdk/Contents/Home", sdkResponse.SdkInfo.Home)

}

func TestSdkService_Run_Go(t *testing.T) {
	manager := endly.New()
	var credential, err = util.GetDummyCredential()
	if err != nil {
		log.Fatal(err)
	}
	target := url.NewResource("ssh://127.0.0.1", credential)
	SSHService, err := exec.GetReplayService("test/go")
	if err != nil {
		log.Fatal(err)
	}
	context, err := exec.OpenTestContext(manager, target, SSHService)
	if err != nil {
		log.Fatal(err)
	}
	sdkResponse := &sdk.SetResponse{}
	sdkRequest, err := sdk.NewSetRequestFromURL("test/go.json")
	if !assert.Nil(t, err) {
		return
	}
	err = endly.Run(context, sdkRequest, sdkResponse)
	if err != nil {
		log.Fatal(err)
	}

	if assert.Nil(t, err) {
		assert.EqualValues(t, "/opt/sdk/go", sdkResponse.SdkInfo.Home)
	}

}
