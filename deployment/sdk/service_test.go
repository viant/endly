package sdk_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/deployment/sdk"
	_ "github.com/viant/endly/static"
	"github.com/viant/endly/system/exec"
	"github.com/viant/endly/system/storage"
	"github.com/viant/endly/util"
	_ "github.com/viant/toolbox/storage/scp"
	"github.com/viant/toolbox/url"
	"log"
	"strings"
	"testing"
)

func TestSdkService_Run_Jdk(t *testing.T) {
	manager := endly.New()
	var credentials, err = util.GetDummyCredential()
	if err != nil {
		log.Fatal(err)
	}
	target := url.NewResource("ssh://127.0.0.1", credentials)
	context, err := exec.NewSSHReplayContext(manager, target, "test/jdk")
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
	var credentials, err = util.GetDummyCredential()
	if err != nil {
		log.Fatal(err)
	}
	target := url.NewResource("ssh://127.0.0.1", credentials)
	context, err := exec.NewSSHReplayContext(manager, target, "test/go")
	if err != nil {
		log.Fatal(err)
	}
	sdkResponse := &sdk.SetResponse{}
	sdkRequest, err := sdk.NewSetRequestFromURL("test/go.yaml")
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

func TestSdkService_Run_Node(t *testing.T) {
	manager := endly.New()
	var credentials, err = util.GetDummyCredential()
	if err != nil {
		log.Fatal(err)
	}
	target := url.NewResource("ssh://127.0.0.1", credentials)
	context, err := exec.NewSSHReplayContext(manager, target, "test/node")
	if err != nil {
		log.Fatal(err)
	}
	mem := storage.UseMemoryService(context)
	mem.Upload("https://nodejs.org/dist/v8.1.1/node-v8.1.1-darwin-x64.tar.gz", strings.NewReader("abc"))

	sdkResponse := &sdk.SetResponse{}
	sdkRequest, err := sdk.NewSetRequestFromURL("test/node.yaml")
	if !assert.Nil(t, err) {
		return
	}

	err = endly.Run(context, sdkRequest, sdkResponse)
	if !assert.Nil(t, err) {
		return
	}
	assert.EqualValues(t, "/opt/sdk/node", sdkResponse.SdkInfo.Home)

}
