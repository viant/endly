package sdk

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/service/deployment/deploy"
	"github.com/viant/endly/internal/util"
	"github.com/viant/endly/model"
	"github.com/viant/endly/service/system/exec"
	"path"
)

// TODO complete implementation
type goService struct{}

func (s *goService) setSdk(context *endly.Context, request *SetRequest) (*Info, error) {
	var result = &Info{}
	var sdkHome = path.Join(request.BaseLocation, "go")
	var runResponse = &exec.RunResponse{}

	setGoROOT := exec.NewRunRequest(request.Target, false, fmt.Sprintf("export GOROOT='%v'", sdkHome))
	_ = endly.Run(context, setGoROOT, nil)

	var extractRequest = exec.NewExtractRequest(request.Target, exec.DefaultOptions(),
		exec.NewExtractCommand("go version", "", nil, nil,
			model.NewExtract("version", "go version go([^\\s]+)", false, false)),
	)

	extractRequest.SystemPaths = append(extractRequest.SystemPaths, fmt.Sprintf("%v/bin", sdkHome))
	if err := endly.Run(context, extractRequest, runResponse); err != nil {
		return nil, err
	}
	var stdout = runResponse.Stdout()
	if util.CheckCommandNotFound(stdout) || util.CheckNoSuchFileOrDirectory(stdout) {
		return nil, errSdkNotFound
	}
	result.Sdk = "go"
	result.Home = sdkHome
	if version, ok := runResponse.Data["version"]; ok {
		result.Version = version.(string)
	}
	if !deploy.MatchVersion(request.Version, result.Version) {
		return nil, errSdkNotFound
	}
	if result.Version == "" {
		result.Version = request.Version
	}
	return result, nil
}
