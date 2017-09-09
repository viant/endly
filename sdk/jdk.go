package sdk

import (
	"fmt"
	"github.com/viant/endly"
)

type jdkService struct {
}

func (s *jdkService) setSdk(context *endly.Context, request *SetSdkRequest) (*SetSdkResponse, error) {
	var response = &SetSdkResponse{}

	commandResponse, err := context.Execute(request.Target, &endly.ManagedCommand{
		Executions: []*endly.Execution{
			{
				Command: fmt.Sprintf("/usr/libexec/java_home -v%v", request.Version),
				Error:   []string{"command not found"},
				Extraction: []*endly.DataExtraction{
					{
						RegExpr:  "(.+jdk.+)",
						StateKey: "JAVA_HOME",
					},
					{
						RegExpr:  "(.+jvm.+)",
						StateKey: "JAVA_HOME",
					},
				},
			},
			{
				Command: "${JAVA_HOME}/bin/java -version",
				Extraction: []*endly.DataExtraction{
					{
						RegExpr: fmt.Sprintf("\"(%v[^\"]+)", request.Version),
						Name:    "build",
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
	_, err = context.Execute(request.Target, &endly.ManagedCommand{
		Executions: []*endly.Execution{
			{
				Command: fmt.Sprintf("export JAVA_HOME='%v'", response.Home),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}
