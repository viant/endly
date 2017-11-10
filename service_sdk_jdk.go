package endly

import (
	"fmt"
	"github.com/lunixbochs/vtclean"
	"strings"
)

type systemJdkService struct{}

func (s *systemJdkService) checkJavaVersion(context *Context, jdkCandidate string, request *SystemSdkSetRequest) (*SystemSdkSetResponse, error) {
	var response = &SystemSdkSetResponse{}
	commandResponse, err := context.Execute(request.Target, &ManagedCommand{



		Executions: []*Execution{
			{
				Command: jdkCandidate + "java -version",
				Extraction: []*DataExtraction{
					{
						RegExpr: "build (\\d\\.\\d).+",
						Key:     "build",
					},
				},
				Error: []string{commandNotFound, noSuchFileOrDirectory, programCanBeFound},
			},
			{
				Command: fmt.Sprintf(jdkCandidate + "jrunscript -e 'java.lang.System.out.println(java.lang.System.getProperty(\"java.home\"));'"),
				Extraction: []*DataExtraction{
					{
						RegExpr: "(.+)",
						Key:     "JAVA_HOME",
					},
				},
				Error: []string{commandNotFound, noSuchFileOrDirectory, programCanBeFound},
			},
		},
	})

	if err != nil {
		return nil, err
	}

	if javaHome, ok := commandResponse.Extracted["JAVA_HOME"]; ok {
		if build, ok := commandResponse.Extracted["build"]; ok {
			if build == request.Version {
				response.Version = build
				response.Home = strings.Replace(javaHome, "/jre", "", 1)
				context.Execute(request.Target, fmt.Sprintf("export JAVA_HOME='%v'", response.Home))


				return response, nil
			}
			return nil, fmt.Errorf("Invalid version was found expected: %v, but had: %v\n", request.Version, build)
		}
	}
	return nil, fmt.Errorf("Failed to check java version")

}

func (s *systemJdkService) getJavaHomeCheckCommand(context *Context, request *SystemSdkSetRequest) string {
	target, err  := context.ExpandResource(request.Target)
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



func (s *systemJdkService) setSdk(context *Context, request *SystemSdkSetRequest) (*SystemSdkSetResponse, error) {
	var response = &SystemSdkSetResponse{}
	if context.Contains(response) {
		var ok bool
		if response, ok = context.GetOptional(response).(*SystemSdkSetResponse); ok {
			if response.Version == request.Version && response.Sdk == request.Sdk && response.SessionID == request.Target.Host() {
				return response, nil
			}
		}
	}
	response, err := s.checkJavaVersion(context, "", request)
	if err == nil {
		return response, nil
	}

	jdkHomeCheckCommand := s.getJavaHomeCheckCommand(context, request)

	commandResponse, err := context.Execute(request.Target, &ManagedCommand{
		Executions: []*Execution{
			{
				Command: jdkHomeCheckCommand,
				Extraction: []*DataExtraction{

					{
						RegExpr: "(.+jdk.+)",
						Key:     "JAVA_HOME",
					},
					{
						RegExpr: "(.+jvm.+)",
						Key:     "JAVA_HOME",
					},
				},
				Error: []string{commandNotFound, noSuchFileOrDirectory, programCanBeFound},
			},
		},
	})

	if home, ok := commandResponse.Extracted["JAVA_HOME"]; ok {
		if strings.Contains(home, "*") {
			return nil, sdkNotFound
		}
		var jdkCandidate = vtclean.Clean(home, false)
		response, err = s.checkJavaVersion(context, jdkCandidate+"/bin/", request)
		if err == nil {
			context.Put(response, response)
			return response, nil
		}
	}
	return nil, sdkNotFound
}
