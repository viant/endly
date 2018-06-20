package web

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"os/exec"
	"testing"
)

func TestService_Run(t *testing.T) {

	output, _ := exec.Command("rm", "-rf", "/tmp/ee").CombinedOutput()
	fmt.Printf("%s\n", output)

	var templateURL = toolbox.URLPathJoin(url.NewResource("../").URL, "template")
	var assetURL = toolbox.URLPathJoin(url.NewResource("../").URL, "asset")
	srv := NewService(templateURL, assetURL)

	resp, err := srv.Run(&RunRequest{
		Datastore: []*Datastore{{
			Driver:            "mysql",
			Name:              "db1",
			Config:            true,
			MultiTableMapping: false,
		},
		},
		Build: &Build{
			Sdk:         "go:1.9",
			Docker:      false,
			App:         "myapp",
			TemplateApp: "go/webdb",
		},
		Testing: &Testing{
			Selenium:    false,
			HTTP:        true,
			REST:        false,
			UseCaseData: "test",
		},
	})
	assert.Nil(t, err)
	assert.NotNil(t, resp)

}

func TestService_Get(t *testing.T) {
	var templateURL = toolbox.URLPathJoin(url.NewResource("../").URL, "template")
	var assetURL = toolbox.URLPathJoin(url.NewResource("../").URL, "asset")
	srv := NewService(templateURL, assetURL)
	resp, err := srv.Get(&GetRequest{})
	assert.Nil(t, err)
	assert.NotNil(t, resp)
}
