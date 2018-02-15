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
	SystemPaths []string          `description:"path that will be appended to the current SSH execution session the current and future commands"`                                 //path that will be added to the system paths
	Terminators []string          `description:"fragment that helps identify that command has been completed - the best is to leave it empty, which is the detected bash prompt"` //fragment that helps identify that command has been completed - the best is to leave it empty, which is the detected bash prompt
	TimeoutMs   int               `description:"time after command was issued for waiting for command output if expect fragment were not matched"`                                //time after command was issued for waiting for command output if expect fragment were not matched.
	Directory   string            `description:"directory where this command should start - if does not exists there is no exception"`                                            //directory where command should run
	Env         map[string]string `description:"environment variables to be set before command runs"`                                                                             //environment variables to be set before command runs
}

//Execution represents an execution instructions
type Execution struct {
	Credentials map[string]string `description:"actual secured credential details as map { '**mysql**': 'path to credential' }like password, etc..., if secure is not empty it will replace **** in command just before execution, to replace user name from credential file use ## prefixed key"` //actual secured credential details as map { '**mysql**': 'path to credentail' }like password, etc..., if secure is not empty it will replace **** in command just before execution
	MatchOutput string            `description:"only run this execution command is output from a previous command is matched"`                                                                                                                                                                     //only run this execution is output from a previous command is matched
	Command     string            `required:"true" description:"shell command to be executed"`                                                                                                                                                                                                     //command to be executed
	Extraction  DataExtractions   `description:"stdout data extraction instruction"`                                                                                                                                                                                                               //Stdout data extraction instruction
	Errors      []string          `description:"fragments that will terminate execution with error if matched with standard output, in most cases leave empty"`                                                                                                                                    //fragments that will terminate execution with error if matched with standard output
	Success     []string          `description:"if specified absence of all of the these fragment will terminate execution with error, in most cases leave empty"`                                                                                                                                 //if specified absence of all of the these fragment will terminate execution with error.
}

//ExtractableCommandRequest represents managed command request
type ExtractableCommandRequest struct {
	SuperUser          bool                `description:"flag to run as super user, in this case sudo will be added to all individual commands unless present, and Target.Credentials password will be used"` ///flag to run it as super user
	Target             *url.Resource       `required:"true" description:"host where command runs" `                                                                                                           //execution target - destination where to run a command.
	ExtractableCommand *ExtractableCommand `description:"command with data extraction instruction "`                                                                                                          //managed command
}

//CommandRequest represents a simple command
type CommandRequest struct {
	SuperUser bool          `description:"flag to run as super user, in this case sudo will be added to all individual commands unless present, and Target.Credentials password will be used"` ///flag to run it as super user
	Target    *url.Resource `required:"true" description:"host where command runs" `                                                                                                           //execution target - destination where to run a command.
	Commands  []string      `required:"true" description:"command list" `                                                                                                                      //list of commands to run
	TimeoutMs int           `description:"time defining how long command can run before timeout, for long running command like large project mvn build, set this value high, otherwise leave empty"`
}

//Validate validates managed command request
func (r *ExtractableCommandRequest) Validate() error {
	if r.Target == nil {
		return fmt.Errorf("execution target was empty")
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
