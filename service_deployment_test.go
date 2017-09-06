package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"testing"
)

func TestNewDeploymentService(t *testing.T) {
	manager := endly.NewServiceManager()
	srv, err := manager.Service(endly.DeploymentServiceId)
	assert.Nil(t, err)
	assert.NotNil(t, srv)

	context := manager.NewContext(toolbox.NewContext())
	defer context.Close()

	response := srv.Run(context, &endly.DeploymenConfig{
		Transfer: &endly.Transfer{
			Source: &endly.Resource{
				URL: "http://mirrors.gigenet.com/apache/maven/maven-3/3.5.0/binaries/apache-maven-3.5.0-bin.tar.gz",
			},
			Target: &endly.Resource{
				Name:    "apache-maven",
				Version: "3.5.0",
				URL:     "scp://127.0.0.1:22/usr/local/",
			},
		},
		VersionCheck: &endly.ManagedCommand{
			Options: &endly.ExecutionOptions{
				SystemPaths: []string{"/usr/local/maven/bin"},
			},
			Executions: []*endly.Execution{
				{
					Command: "mvn -version",
					Extraction: []*endly.DataExtraction{{
						Name:    "Version",
						RegExpr: "Apache Maven (\\d+\\.\\d+\\.\\d+)",
					},
					},
				},
			},
		},
		After: &endly.ManagedCommand{
			Options: &endly.ExecutionOptions{
				Directory: "/urs/local",
			},
			Executions: []*endly.Execution{
				{
					Command: "tar xvzf apache-maven-3.5.0-bin.tar.gz",
					Error:   []string{"Error"},
				},
				{
					Command: "mv apache-maven-3.5.0 maven",
					Error:   []string{"No"},
				},
			},
		},
	})

	assert.Nil(t, response.Error)

}
