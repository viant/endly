package sdk

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/system/exec"
	"github.com/viant/endly/util"
	"os"
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
		_ = endly.Run(context, exec.NewRunRequest(request.Target, false, fmt.Sprintf("export GOPATH='%v'", goPath)), nil)
	}

	var extractRequest = exec.NewExtractRequest(request.Target, exec.DefaultOptions(),
		exec.NewExtractCommand("export GOROOT='/opt/sdk/go'", "", nil, nil),
		exec.NewExtractCommand("go version", "", nil, nil,
			endly.NewExtract("version", "go version go([^\\s]+)", false)),
	)
	runResponse := &exec.RunResponse{}
	if err := endly.Run(context, extractRequest, runResponse); err != nil {
		return nil, err
	}
	var stdout = runResponse.Stdout()
	if util.CheckCommandNotFound(stdout) || util.CheckNoSuchFileOrDirectory(stdout) {
		stdout = runResponse.Stdout()
		if util.CheckCommandNotFound(stdout) || util.CheckNoSuchFileOrDirectory(stdout) {
			return nil, errSdkNotFound
		}
	}
	result.Sdk = "go"
	result.Home = "/opt/sdk/go"
	result.Version = runResponse.Extracted["version"]
	if result.Version == "" {
		result.Version = request.Version
	}
	return result, nil
}
