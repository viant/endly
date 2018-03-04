package sdk

import (
	"fmt"
	"os"
	"github.com/viant/endly"
	"github.com/viant/endly/system/exec"
	"github.com/viant/endly/util"
)

//TODO complete implementation
type goService struct{}

func (s *goService) setSdk(context *endly.Context, request *SetRequest) (*Info, error) {
	var result = &Info{}

	goPath, ok := request.Env["GOPATH"]
	if !ok || goPath == "" {
		goPath = os.Getenv("GOPATH")
	}

	if goPath != "" {
		exec.Execute(context, request.Target, fmt.Sprintf("export GOPATH='%v'", goPath))
	}

	commandResponse, err := exec.Execute(context, request.Target, &exec.ExtractableCommand{
		Executions: []*exec.Execution{
			{
				Command: "export GOROOT='/opt/sdk/go'",
			},
			{
				Command: "go version",
				Extraction: []*endly.DataExtraction{
					{
						RegExpr: "go version go([^\\s]+)",
						Key:     "version",
					},
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	var stdout = commandResponse.Stdout()
	if util.CheckCommandNotFound(stdout) || util.CheckNoSuchFileOrDirectory(stdout) {
		stdout = commandResponse.Stdout()
		if util.CheckCommandNotFound(stdout) || util.CheckNoSuchFileOrDirectory(stdout) {
			return nil, errSdkNotFound
		}
	}
	if err != nil {
		return nil, err
	}
	result.Sdk = "go"
	result.Version = commandResponse.Extracted["version"]
	if result.Version == "" {
		result.Version = request.Version
	}
	return result, nil
}
