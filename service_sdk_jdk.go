package endly

import (
	"fmt"
	"github.com/lunixbochs/vtclean"
	"strings"
)

type systemJdkService struct{}

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
	commandResponse, err := context.Execute(request.Target, &ManagedCommand{
		Executions: []*Execution{
			{
				Command: fmt.Sprintf("/usr/libexec/java_home -v%v", request.Version),
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
			},
			{
				Command: "${JAVA_HOME}/bin/java -version",
				Extraction: []*DataExtraction{
					{
						RegExpr: fmt.Sprintf("\"(%v[^\"]+)", request.Version),
						Key:     "build",
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
		var version = request.Version
		version = strings.Replace(version, "1.", "", 1)
		commandResponse, err = context.Execute(request.Target, &ManagedCommand{
			Executions: []*Execution{
				{

					Command: fmt.Sprintf("update-java-alternatives --list | grep %v", request.Version),
					Extraction: []*DataExtraction{
						{
							RegExpr: "(/[^\\s]+jvm[^\\s]+)",
							Key:     "JAVA_HOME",
						},
					},
				},
				{

					Command: fmt.Sprintf("update-java-alternatives --list | grep %v", version),
					Extraction: []*DataExtraction{
						{
							RegExpr: "(/[^\\s]+jvm[^\\s]+)",
							Key:     "JAVA_HOME",
						},
					},
				},
			},
		})

		if javaHome, ok := commandResponse.Extracted["JAVA_HOME"]; ok {
			var candidate = vtclean.Clean(javaHome, false)
			if strings.Contains(candidate, "*") {
				return nil, sdkNotFound
			}
			response.Home = candidate
			commandResponse, err = context.Execute(request.Target, &ManagedCommand{
				Executions: []*Execution{
					{
						Command: response.Home + "/bin/java -version",
						Extraction: []*DataExtraction{
							{
								RegExpr: fmt.Sprintf("\"(%v[^\"]+)", request.Version),
								Key:     "build",
							},
						},
					},
				},
			})
			stdout = commandResponse.Stdout()
			if CheckCommandNotFound(stdout) || CheckNoSuchFileOrDirectory(stdout) {
				return nil, sdkNotFound
			}
		}
	}

	if build, ok := commandResponse.Extracted["build"]; ok {
		response.Build = build
	}

	if response.Build == "" {
		return nil, sdkNotFound
	}

	if home, ok := commandResponse.Extracted["JAVA_HOME"]; ok {
		response.Home = vtclean.Clean(home, false)

	}

	_, err = context.Execute(request.Target, &ManagedCommand{
		Executions: []*Execution{
			{
				Command: fmt.Sprintf("export JAVA_HOME='%v'", response.Home),
			},
		},
	})

	if err != nil {
		return nil, err
	}
	context.Put(response, response)
	return response, nil
}
