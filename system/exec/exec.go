package exec

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/toolbox/url"
)

//Execute execute shell command
func Execute(context *endly.Context, target *url.Resource, command interface{}) (*RunResponse, error) {
	if command == nil {
		return nil, nil
	}
	var err error
	var commandRequest *ExtractRequest
	switch actualCommand := command.(type) {
	case *SuperRunRequest:
		actualCommand.Target = target
		commandRequest, err = actualCommand.AsExtractRequest(context)
		if err != nil {
			return nil, err
		}
	case *RunRequest:
		actualCommand.Target = target
		commandRequest = actualCommand.AsExtractRequest()
	case *ExtractableCommand:
		commandRequest = NewExtractRequest(target, actualCommand)
	case string:
		request := RunRequest{
			Target:   target,
			Commands: []string{actualCommand},
		}
		commandRequest = request.AsExtractRequest()
	case []string:
		request := RunRequest{
			Target:   target,
			Commands: actualCommand,
		}
		commandRequest = request.AsExtractRequest()

	default:
		return nil, fmt.Errorf("unsupported command: %T", command)
	}
	execService, err := context.Service(ServiceID)
	if err != nil {
		return nil, err
	}
	response := execService.Run(context, commandRequest)
	if response.Err != nil {
		return nil, response.Err
	}
	if commandResult, ok := response.Response.(*RunResponse); ok {
		return commandResult, nil
	}
	return nil, nil
}

//ExecuteAsSuperUser executes command as super user
func ExecuteAsSuperUser(context *endly.Context, target *url.Resource, command *ExtractableCommand) (*RunResponse, error) {
	superUserRequest := SuperRunRequest{
		Target:        target,
		MangedCommand: command,
	}
	request, err := superUserRequest.AsExtractRequest(context)
	if err != nil {
		return nil, err
	}
	return Execute(context, target, request.ExtractableCommand)
}
