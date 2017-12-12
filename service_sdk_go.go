package endly

import (
	"fmt"
)

//TODO complete implementation
type systemGoService struct{}

func (s *systemGoService) setSdk(context *Context, request *SystemSdkSetRequest) (*SystemSdkInfo, error) {
	var result = &SystemSdkInfo{}


	if goPath, ok := request.Env["GOPATH"]; ok {
		context.Execute(request.Target, fmt.Sprintf("export GOPATH='%v'", goPath))
	}
	commandResponse, err := context.Execute(request.Target, &ExtractableCommand{
		Executions: []*Execution{
			{
				Command: "export GOROOT='/opt/sdk/go'",
			},
			{
				Command: "go version",
				Extraction: []*DataExtraction{
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
	if CheckCommandNotFound(stdout) || CheckNoSuchFileOrDirectory(stdout) {
		stdout = commandResponse.Stdout()
		if CheckCommandNotFound(stdout) || CheckNoSuchFileOrDirectory(stdout) {
			return nil, sdkNotFound
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
