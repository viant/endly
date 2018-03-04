package exec

import (
	"github.com/viant/endly"
	"github.com/viant/toolbox/url"
	"fmt"
	"strings"
	"github.com/viant/toolbox/ssh"
	"github.com/viant/endly/util"
	"github.com/pkg/errors"
)


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
	Credentials map[string]string     `description:"actual secured credential details as map { '**mysql**': 'path to credential' }like password, etc..., if secure is not empty it will replace **** in command just before execution, to replace user name from credential file use ## prefixed key"` //actual secured credential details as map { '**mysql**': 'path to credentail' }like password, etc..., if secure is not empty it will replace **** in command just before execution
	MatchOutput string                `description:"only run this execution command is output from a previous command is matched"`                                                                                                                                                                     //only run this execution is output from a previous command is matched
	Command     string                `required:"true" description:"shell command to be executed"`                                                                                                                                                                                                     //command to be executed
	Extraction  endly.DataExtractions `description:"stdout data extraction instruction"`                                                                                                                                                                                                               //Stdout data extraction instruction
	Errors      []string              `description:"fragments that will terminate execution with error if matched with standard output, in most cases leave empty"`                                                                                                                                    //fragments that will terminate execution with error if matched with standard output
	Success     []string              `description:"if specified absence of all of the these fragment will terminate execution with error, in most cases leave empty"`                                                                                                                                 //if specified absence of all of the these fragment will terminate execution with error.
}


//ExtractableCommand represent managed command, to execute and extract data, detect success or error state
type ExtractableCommand struct {
	Options    *ExecutionOptions `description:"execution option like system paths, env, starting directory"`
	Executions []*Execution      `description:"execution commands"` //actual execution instruction
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

//CommandLog represents an executed command with Stdin, Stdout or Error
type CommandLog struct {
	Stdin  string
	Stdout string
	Error  string
}

//CommandResponse represents a command response with logged commands.
type CommandResponse struct {
	Session   string
	Commands  []*CommandLog
	Extracted map[string]string
	Error     string
}

//OpenSessionRequest represents an open session request.
type OpenSessionRequest struct {
	Target          *url.Resource      //Session is created from target host (servername, port)
	Config          *ssh.SessionConfig //ssh configuration
	SystemPaths     []string           //system path that are applied to the ssh session
	Env             map[string]string
	Transient       bool        //if this flag is true, caller is responsible for closing session, othewise session is closed as context is closed
	CommandsBasedir string      //capture all ssh service command in supplied dir (for unit test only)
	ReplayService   ssh.Service //use Ssh ReplayService instead of actual SSH service (for unit test only)
}


func (r *OpenSessionRequest) Validate() error {
	if r.Target == nil {
		return errors.New("target was empty")
	}
	return nil
}

//OpenSessionResponse represents a session id
type OpenSessionResponse struct {
	SessionID string
}

//CloseSessionRequest closes session
type CloseSessionRequest struct {
	SessionID string
}

//CloseSessionResponse closes session response
type CloseSessionResponse struct {
	SessionID string
}

//SuperUserCommandRequest represents a super user command,
type SuperUserCommandRequest struct {
	Target        *url.Resource       //target destination where to run a command.
	MangedCommand *ExtractableCommand //managed command
}

//Validate validates managed command request
func (r *ExtractableCommandRequest) Validate() error {
	if r.Target == nil {
		return fmt.Errorf("execution target was empty")
	}
	if r.ExtractableCommand == nil {
		return fmt.Errorf("extractableCommand was empty")
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
			Errors:  []string{util.CommandNotFound, util.NoSuchFileOrDirectory, util.ErrorIsNotRecoverable},
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

//Add appends provided log into commands slice.
func (i *CommandResponse) Add(log *CommandLog) {
	if len(i.Commands) == 0 {
		i.Commands = make([]*CommandLog, 0)
	}
	i.Commands = append(i.Commands, log)
}

//Stdout returns stdout for provided index, or all concatenated otherwise
func (i *CommandResponse) Stdout(indexes ...int) string {
	if len(indexes) == 0 {
		var result = make([]string, len(i.Commands))
		for j, stream := range i.Commands {
			result[j] = stream.Stdout
		}
		return strings.Join(result, "\r\n")
	}
	var result = make([]string, len(indexes))
	for _, index := range indexes {
		if index < len(i.Commands) {
			result = append(result, i.Commands[index].Stdout)
		}
	}
	return strings.Join(result, "\r\n")
}

//NewCommandResponse creates a new CommandResponse
func NewCommandResponse(session string) *CommandResponse {
	return &CommandResponse{
		Session:   session,
		Commands:  make([]*CommandLog, 0),
		Extracted: make(map[string]string),
	}
}

//NewCommandLog creates a new command log
func NewCommandLog(stdin, stdout string, err error) *CommandLog {
	result := &CommandLog{
		Stdin: stdin,
	}
	if err != nil {
		result.Error = fmt.Sprintf("%v", err)
	} else {
		result.Stdout = stdout
	}
	return result
}

//AsCommandRequest returns ExtractableCommandRequest
func (r *SuperUserCommandRequest) AsCommandRequest(context *endly.Context) (*ExtractableCommandRequest, error) {
	target, err := context.ExpandResource(r.Target)
	if err != nil {
		return nil, err
	}
	var result = &ExtractableCommandRequest{
		Target: target,
		ExtractableCommand: &ExtractableCommand{
			Executions: make([]*Execution, 0),
		},
	}
	var executionOptions = &ExecutionOptions{
		Terminators: []string{"Password", util.CommandNotFound},
	}
	if r.MangedCommand == nil {
		return nil, fmt.Errorf("command was ampty")
	}
	if r.MangedCommand.Options != nil {
		executionOptions.Terminators = append(executionOptions.Terminators, r.MangedCommand.Options.Terminators...)
		executionOptions.TimeoutMs = r.MangedCommand.Options.TimeoutMs
		executionOptions.Directory = r.MangedCommand.Options.Directory
		executionOptions.SystemPaths = r.MangedCommand.Options.SystemPaths
	}
	result.ExtractableCommand.Options = executionOptions
	var errors = make([]string, 0)
	var extractions = make([]*endly.DataExtraction, 0)

	var credentials = make(map[string]string)

	for _, execution := range r.MangedCommand.Executions {
		if execution.Command == "" {
			continue
		}
		if len(execution.Credentials) > 0 {
			for k, v := range execution.Credentials {
				credentials[k] = v
			}
		}
		sudo := ""
		if len(execution.Command) > 1 && !strings.Contains(execution.Command, "sudo") {
			sudo = "sudo "
		}
		newExecution := &Execution{
			Command:     sudo + execution.Command,
			Errors:      execution.Errors,
			Extraction:  execution.Extraction,
			Success:     execution.Success,
			MatchOutput: execution.MatchOutput,
			Credentials: execution.Credentials,
		}
		if len(execution.Errors) > 0 {
			errors = append(errors, execution.Errors...)
		}
		if len(execution.Extraction) > 0 {
			extractions = append(extractions, execution.Extraction...)
		}
		result.ExtractableCommand.Executions = append(result.ExtractableCommand.Executions, newExecution)
	}

	if target.Credential == "" {
		return nil, fmt.Errorf("Can not run as superuser, credential were empty for target: %v", target.URL)
	}
	credentials[SudoCredentialKey] = target.Credential
	execution := &Execution{
		Credentials: credentials,
		MatchOutput: "Password",
		Command:     SudoCredentialKey,
		Errors:      []string{"Password", util.CommandNotFound},
		Extraction:  extractions,
	}
	execution.Errors = append(execution.Errors, errors...)
	result.ExtractableCommand.Executions = append(result.ExtractableCommand.Executions, execution)
	return result, nil
}
