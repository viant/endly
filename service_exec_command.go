package endly

import (
	"fmt"
	"github.com/viant/toolbox/url"
)

//ExtractableCommand represent managed command, to execute and extract data, detect success or error state
type ExtractableCommand struct {
	Options    *ExecutionOptions //ExecutionOptions
	Executions []*Execution      //actual execution instruction
}

//ExecutionOptions represents an execution options
type ExecutionOptions struct {
	SystemPaths []string          //path that will be added to the system paths
	Terminators []string          //fragment that helps identify that command has been completed - the best is to leave it empty, which is the detected bash prompt
	TimeoutMs   int               //time after command was issued for waiting for command output if expect fragment were not matched.
	Directory   string            //directory where command should run
	Env         map[string]string //environment variables to be set before command runs
}

//Execution represents an execution instructions
type Execution struct {
	Credentials map[string]string //actual secured credential details as map { '**mysql**': 'path to credentail' }like password, etc..., if secure is not empty it will replace **** in command just before execution
	MatchOutput string            //only run this execution is output from a previous command is matched
	Command     string            //command to be executed
	Extraction  DataExtractions   //Stdout data extraction instruction
	Errors      []string          //fragments that will terminate execution with error if matched with standard output
	Success     []string          //if specified absence of all of the these fragment will terminate execution with error.
}

//ExtractableCommandRequest represents managed command request
type ExtractableCommandRequest struct {
	SuperUser          bool                ///flag to run it as super user
	Target             *url.Resource       //target destination where to run a command.
	ExtractableCommand *ExtractableCommand //managed command
}

//CommandRequest represents a simple command
type CommandRequest struct {
	SuperUser bool          //flag is command needs to run as super suer
	Target    *url.Resource //target destination where to run a command.
	Commands  []string      //list of commands to run
	TimeoutMs int
}

//Validate validates managed command request
func (r *ExtractableCommandRequest) Validate() error {
	if r.Target == nil {
		return fmt.Errorf("Target was empty")
	}
	if r.ExtractableCommand == nil {
		return fmt.Errorf("ExtractableCommand was empty")
	}
	return nil
}

//AsExtractableCommandRequest returns ExtractableCommandRequest for this requests
func (r *CommandRequest) AsExtractableCommandRequest() *ExtractableCommandRequest {
	var extractableCommand = &ExtractableCommand{
		Options:    NewExecutionOptions(),
		Executions: make([]*Execution, 0),
	}
	if r.TimeoutMs > 0 {
		extractableCommand.Options.TimeoutMs = r.TimeoutMs
	}
	for _, command := range r.Commands {
		extractableCommand.Executions = append(extractableCommand.Executions, &Execution{
			Command: command,
			Errors:  []string{commandNotFound, noSuchFileOrDirectory, errorIsNotRecoverable},
		})
	}
	return &ExtractableCommandRequest{
		SuperUser:          r.SuperUser,
		Target:             r.Target,
		ExtractableCommand: extractableCommand,
	}
}

//NewExtractableCommandRequest returns a new command request
func NewExtractableCommandRequest(target *url.Resource, execution *ExtractableCommand) *ExtractableCommandRequest {
	return &ExtractableCommandRequest{
		Target:             target,
		ExtractableCommand: execution,
	}
}

//NewSimpleCommandRequest a simple version of ExtractableCommandRequest
func NewSimpleCommandRequest(target *url.Resource, commands ...string) *ExtractableCommandRequest {
	var result = &ExtractableCommandRequest{
		Target: target,
		ExtractableCommand: &ExtractableCommand{
			Executions: make([]*Execution, 0),
		},
	}
	for _, command := range commands {
		result.ExtractableCommand.Executions = append(result.ExtractableCommand.Executions, &Execution{
			Command: command,
		})
	}
	return result
}

//NewExecutionOptions creates a new execution options
func NewExecutionOptions() *ExecutionOptions {
	return &ExecutionOptions{
		SystemPaths: make([]string, 0),
		Terminators: make([]string, 0),
		Env:         make(map[string]string),
	}
}
