package endly

import (
	"fmt"
)

//TODO complete implementation
type systemGoService struct{}

func (s *systemGoService) setSdk(context *Context, request *SystemSdkSetRequest) (*SystemSdkSetResponse, error) {
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
				Command: fmt.Sprintf("go version"),
				Extraction: []*DataExtraction{
					{
						RegExpr: fmt.Sprintf("go version go([^\\s]+)"),
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
	context.Put(response, response)
	return response, nil
}
