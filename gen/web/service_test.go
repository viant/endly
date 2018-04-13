package web

import (
	"testing"
	"github.com/viant/toolbox/url"
	"github.com/stretchr/testify/assert"
	"os/exec"
	"fmt"
	"github.com/viant/toolbox"
)

func TestService_Run(t *testing.T) {

	output, _ := exec.Command("rm", "-rf", "/tmp/ee").CombinedOutput()
	fmt.Printf("%s\n", output)



	srv := NewService(url.NewResource("../").URL)


	resp, err := srv.Run(&RunRequest{
		Datastore: &Datastore{
			Driver:      "mysql",
			Name:        "db1",
			Config:      true,

		},
		Build: &Build{
			Sdk:         "go:1.9",
			Docker:      false,
			App:         "myapp",
			TemplateApp: "go/webdb",
		},
		Testing: &Testing{
			Selenium:    true,
			HTTP:        true,
			REST:        true,
			UseCaseData: true,

		},
	})
	assert.Nil(t, err)
	assert.NotNil(t, resp)

}


func TestService_Get(t *testing.T) {
	srv := NewService(url.NewResource("../").URL)
	resp, err := srv.Get(&GetRequest{})
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	toolbox.Dump(resp)
}
