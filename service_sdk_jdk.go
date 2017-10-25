package endly

import (
	"fmt"
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
				Error:   []string{"command not found"},
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

	if home, ok := commandResponse.Extracted["JAVA_HOME"]; ok {
		response.Home = home
	}

	if build, ok := commandResponse.Extracted["build"]; ok {
		response.Build = build

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
