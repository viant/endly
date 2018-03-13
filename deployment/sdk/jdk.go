package sdk

import (
	"fmt"
	"github.com/lunixbochs/vtclean"
	"github.com/pkg/errors"
	"github.com/viant/endly"
	"github.com/viant/endly/system/exec"
	"github.com/viant/endly/util"
	"strings"
)

type jdkService struct{}

func (s *jdkService) checkJavaVersion(context *endly.Context, jdkCandidate string, request *SetRequest) (*Info, error) {
	var result = &Info{}

	extractRequest := exec.NewExtractRequest(request.Target, exec.DefaultOptions(),
		exec.NewExtractCommand(jdkCandidate+"java -version", "", nil,
			util.StdErrors,
			endly.NewExtract("build", "build (\\d\\.\\d).+", false)),
		exec.NewExtractCommand(fmt.Sprintf(jdkCandidate+"jrunscript -e 'java.lang.System.out.println(java.lang.System.getProperty(\"java.home\"));'"), "", nil,
			util.StdErrors,
			endly.NewExtract("JAVA_HOME", "(.+)", false)))

	commandResponse := &exec.RunResponse{}
	if err := endly.Run(context, extractRequest, commandResponse); err != nil {
		return nil, err
	}
	if javaHome, ok := commandResponse.Extracted["JAVA_HOME"]; ok {
		if build, ok := commandResponse.Extracted["build"]; ok {
			if build == request.Version {
				result.Version = build
				result.Home = strings.Replace(javaHome, "/jre", "", 1)
				endly.Run(context, exec.NewRunRequest(request.Target, false, fmt.Sprintf("export JAVA_HOME='%v'", result.Home)), nil)
				return result, nil
			}
			return nil, fmt.Errorf("invalid version was found expected: %v, but had: %v", request.Version, build)
		}
	}
	return nil, errors.New("failed to check java version")

}

func (s *jdkService) getJavaHomeCheckCommand(context *endly.Context, request *SetRequest) string {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return ""
	}
	operatingSystem := context.OperatingSystem(target.Host())
	if operatingSystem.System == "darwin" {
		return fmt.Sprintf("/usr/libexec/java_home -v%v", request.Version)
	}
	var version = request.Version
	version = strings.Replace(version, "1.", "", 1)
	return fmt.Sprintf("update-java-alternatives --list | grep %v", version)
}

func (s *jdkService) setSdk(context *endly.Context, request *SetRequest) (*Info, error) {
	var result = &Info{}
	result, err := s.checkJavaVersion(context, "", request)
	if err == nil {
		return result, nil
	}
	jdkHomeCheckCommand := s.getJavaHomeCheckCommand(context, request)
	extractRequest := exec.NewExtractRequest(request.Target, exec.DefaultOptions(),
		exec.NewExtractCommand(jdkHomeCheckCommand, "", nil, util.StdErrors,
			endly.NewExtract("JAVA_HOME", "(.+jdk.+)", false),
			endly.NewExtract("JAVA_HOME", "(.+jvm.+)", false)))

	commandResponse := &exec.RunResponse{}
	if err := endly.Run(context, extractRequest, commandResponse); err != nil {
		return nil, err
	}
	if home, ok := commandResponse.Extracted["JAVA_HOME"]; ok {
		if strings.Contains(home, "*") {
			return nil, errSdkNotFound
		}
		var jdkCandidate = vtclean.Clean(home, false)
		result, err = s.checkJavaVersion(context, jdkCandidate+"/bin/", request)
		if err == nil {
			return result, nil
		}
	}
	return nil, errSdkNotFound
}
