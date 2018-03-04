package exec

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/toolbox/url"
)

//Execute execute shell command
func Execute(context *endly.Context, target *url.Resource, command interface{}) (*CommandResponse, error) {
	if command == nil {
		return nil, nil
	}
	var err error
	var commandRequest *ExtractableCommandRequest
	switch actualCommand := command.(type) {
	case *SuperUserCommandRequest:
		actualCommand.Target = target
		commandRequest, err = actualCommand.AsCommandRequest(context)
		if err != nil {
			return nil, err
		}
	case *CommandRequest:
		actualCommand.Target = target
		commandRequest = actualCommand.AsExtractableCommandRequest()
	case *ExtractableCommand:
		commandRequest = NewExtractableCommandRequest(target, actualCommand)
	case string:
		request := CommandRequest{
			Target:   target,
			Commands: []string{actualCommand},
		}
		commandRequest = request.AsExtractableCommandRequest()
	case []string:
		request := CommandRequest{
			Target:   target,
			Commands: actualCommand,
		}
		commandRequest = request.AsExtractableCommandRequest()

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
	if commandResult, ok := response.Response.(*CommandResponse); ok {
		return commandResult, nil
	}
	return nil, nil
}

//ExecuteAsSuperUser executes command as super user
func ExecuteAsSuperUser(context *endly.Context, target *url.Resource, command *ExtractableCommand) (*CommandResponse, error) {
	superUserRequest := SuperUserCommandRequest{
		Target:        target,
		MangedCommand: command,
	}
	request, err := superUserRequest.AsCommandRequest(context)
	if err != nil {
		return nil, err
	}
	return Execute(context, target, request.ExtractableCommand)
}
