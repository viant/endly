package endly

import (
	"fmt"
	"github.com/viant/toolbox/url"
)

//ManagedCommand represent managed command, to execute and extract data, detect success or error state
type ManagedCommand struct {
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
	Secure string //actual secured details like password, etc..., if secure is not empty it will replace **** in command just before execution
	//for security Execution message should not be log in.
	MatchOutput string          //only run this execution is output from a previous command is matched
	Command     string          //command to be executed
	Extraction  DataExtractions //Stdout data extraction instruction
	Error       []string        //fragments that will terminate execution with error if matched with standard output
	Success     []string        //if specified absence of all of the these fragment will terminate execution with error.
}

//ManagedCommandRequest represents managed command request
type ManagedCommandRequest struct {
	SuperUser      bool            ///flag to run it as super user
	Target         *url.Resource   //target destination where to run a command.
	ManagedCommand *ManagedCommand //managed command
}

//CommandRequest represents a simple command
type CommandRequest struct {
	SuperUser bool          //flag is command needs to run as super suer
	Target    *url.Resource //target destination where to run a command.
	Commands  []string      //list of commands to run
}

//Validate validates managed command request
func (r *ManagedCommandRequest) Validate() error {
	if r.Target == nil {
		return fmt.Errorf("Target was empty")
	}
	if r.ManagedCommand == nil {
		return fmt.Errorf("ManagedCommand was empty")
	}
	return nil
}

//AsManagedCommandRequest returns ManagedCommandRequest for this requests
func (r *CommandRequest) AsManagedCommandRequest() *ManagedCommandRequest {
	var managedCommand = &ManagedCommand{
		Executions: make([]*Execution, 0),
	}
	for _, command := range r.Commands {
		managedCommand.Executions = append(managedCommand.Executions, &Execution{
			Command: command,
			Error:   []string{commandNotFound, noSuchFileOrDirectory},
		})
	}
	return &ManagedCommandRequest{
		SuperUser:      r.SuperUser,
		Target:         r.Target,
		ManagedCommand: managedCommand,
	}
}

//NewManagedCommandRequest returns a new command request
func NewManagedCommandRequest(target *url.Resource, execution *ManagedCommand) *ManagedCommandRequest {
	return &ManagedCommandRequest{
		Target:         target,
		ManagedCommand: execution,
	}
}

//NewSimpleCommandRequest a simple version of ManagedCommandRequest
func NewSimpleCommandRequest(target *url.Resource, commands ...string) *ManagedCommandRequest {
	var result = &ManagedCommandRequest{
		Target: target,
		ManagedCommand: &ManagedCommand{
			Executions: make([]*Execution, 0),
		},
	}
	for _, command := range commands {
		result.ManagedCommand.Executions = append(result.ManagedCommand.Executions, &Execution{
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
