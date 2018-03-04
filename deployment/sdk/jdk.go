package sdk

import (
	"fmt"
	"github.com/lunixbochs/vtclean"
	"github.com/pkg/errors"
	"strings"
	"github.com/viant/endly"
	"github.com/viant/endly/system/exec"
	"github.com/viant/endly/util"
)

type jdkService struct{}

func (s *jdkService) checkJavaVersion(context *endly.Context, jdkCandidate string, request *SetRequest) (*Info, error) {
	var result = &Info{}
	commandResponse, err := exec.Execute(context, request.Target, &exec.ExtractableCommand{
		Executions: []*exec.Execution{
			{
				Command: jdkCandidate + "java -version",
				Extraction: []*endly.DataExtraction{
					{
						RegExpr: "build (\\d\\.\\d).+",
						Key:     "build",
					},
				},
				Errors: util.StdErrors,
			},
			{
				Command: fmt.Sprintf(jdkCandidate + "jrunscript -e 'java.lang.System.out.println(java.lang.System.getProperty(\"java.home\"));'"),
				Extraction: []*endly.DataExtraction{
					{
						RegExpr: "(.+)",
						Key:     "JAVA_HOME",
					},
				},
				Errors: util.StdErrors,
			},
		},
	})

	if err != nil {
		return nil, err
	}

	if javaHome, ok := commandResponse.Extracted["JAVA_HOME"]; ok {
		if build, ok := commandResponse.Extracted["build"]; ok {
			if build == request.Version {
				result.Version = build
				result.Home = strings.Replace(javaHome, "/jre", "", 1)
				exec.Execute(context, request.Target, fmt.Sprintf("export JAVA_HOME='%v'", result.Home))

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
	commandResponse, err := exec.Execute(context, request.Target, &exec.ExtractableCommand{
		Executions: []*exec.Execution{
			{
				Command: jdkHomeCheckCommand,
				Extraction: []*endly.DataExtraction{

					{
						RegExpr: "(.+jdk.+)",
						Key:     "JAVA_HOME",
					},
					{
						RegExpr: "(.+jvm.+)",
						Key:     "JAVA_HOME",
					},
				},
				Errors: util.StdErrors,
			},
		},
	})

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
