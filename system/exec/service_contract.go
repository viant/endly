package exec

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/viant/endly/model"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/secret"
	"github.com/viant/toolbox/ssh"
	"github.com/viant/toolbox/url"
	"strings"
)

var CommandErrors = []string{util.CommandNotFound, util.NoSuchFileOrDirectory, util.ErrorIsNotRecoverable}

//Options represents an execution options
type Options struct {
	SystemPaths []string          `description:"path that will be appended to the current SSH execution session the current and future commands"`                                                //path that will be added to the system paths
	Terminators []string          `description:"fragment that helps identify that command has been completed - the best is to leave it empty, which is the detected bash prompt"`                //fragment that helps identify that command has been completed - the best is to leave it empty, which is the detected bash prompt
	Errors      []string          `description:"fragments that will terminate execution with error if matched with standard output, in most cases leave empty"`                                  //fragments that will terminate execution with error if matched with standard output
	TimeoutMs   int               `description:"time after command was issued for waiting for command output if expect fragment were not matched"`                                               //time after command was issued for waiting for command output if expect fragment were not matched.
	Directory   string            `description:"directory where this command should start - if does not exists there is no exception"`                                                           //directory where command should run
	Env         map[string]string `description:"environment variables to be set before command runs"`                                                                                            //environment variables to be set before command runs
	SuperUser   bool              `description:"flag to run as super user, in this case sudo will be added to all individual commands unless present, and Target.Secrets password will be used"` ///flag to run it as super user
	Secrets     secret.Secrets    `description:"secrets map see https://github.com/viant/toolbox/tree/master/secret"`
}

//DefaultOptions creates a default execution options
func DefaultOptions() *Options {
	return &Options{
		SystemPaths: make([]string, 0),
		Terminators: make([]string, 0),
		Env:         make(map[string]string),
	}
}

func NewOptions(secrets, env map[string]string, terminators, path []string, superUser bool) *Options {
	if len(terminators) == 0 {
		terminators = []string{}
	}
	if len(terminators) == 0 {
		path = []string{}
	}
	if len(env) == 0 {
		env = make(map[string]string)
	}
	return &Options{
		Env:         env,
		Terminators: terminators,
		SystemPaths: path,
		SuperUser:   superUser,
		Secrets:     secret.NewSecrets(secrets),
	}
}

//Extracts represents an execution instructions
type ExtractCommand struct {
	When       string         `description:"only run this command is criteria is matched i.e $stdout:/password/"`                                              //only run this execution is output from a previous command is matched
	Command    string         `required:"true" description:"shell command to be executed"`                                                                     //command to be executed
	Extraction model.Extracts `description:"stdout data extraction instruction"`                                                                               //Stdout data extraction instruction
	Errors     []string       `description:"fragments that will terminate execution with error if matched with standard output, in most cases leave empty"`    //fragments that will terminate execution with error if matched with standard output
	Success    []string       `description:"if specified absence of all of the these fragment will terminate execution with error, in most cases leave empty"` //if specified absence of all of the these fragment will terminate execution with error.
}

func (c *ExtractCommand) Init() error {
	if strings.TrimSpace(c.When) != "" {
		if strings.Index(c.When, "$") == -1 {
			c.When = fmt.Sprintf("$stdout :/%v/", c.When)
		}
	}
	return nil
}

//Validate validates managed command request
func (r *ExtractRequest) Validate() error {
	if r.Target == nil {
		return fmt.Errorf("target was empty")
	}
	if r.Commands == nil {
		return fmt.Errorf("commands were empty")
	}
	return nil
}

//NewExtractCommand creates a new extract command
func NewExtractCommand(command, when string, success, errors []string, extractions ...*model.Extract) *ExtractCommand {
	if len(success) == 0 {
		success = []string{}
	}
	if len(errors) == 0 {
		errors = []string{}
	}
	return &ExtractCommand{
		Command:    command,
		When:       when,
		Extraction: extractions,
		Success:    success,
		Errors:     errors,
	}
}

//ExtractRequest represents managed command request
type ExtractRequest struct {
	Target *url.Resource `required:"true" description:"host where command runs" ` //execution target - destination where to run a command.
	*Options
	Commands []*ExtractCommand `description:"command with data extraction instruction "` //extract command
}

//Init initialises request
func (r *ExtractRequest) Init() error {
	if r.Options == nil {
		r.Options = DefaultOptions()
	}
	if len(r.Commands) > 0 {
		for _, command := range r.Commands {
			if err := command.Init(); err != nil {
				return err
			}
		}
	}
	return nil
}

//Clones clones requst with supplide target
func (r *ExtractRequest) Clone(target *url.Resource) *ExtractRequest {
	if target == nil {
		target = r.Target
	}
	return &ExtractRequest{
		Target:   target,
		Options:  r.Options,
		Commands: r.Commands,
	}
}

//NewExtractRequest returns a new command request
func NewExtractRequest(target *url.Resource, options *Options, commands ...*ExtractCommand) *ExtractRequest {
	return &ExtractRequest{
		Target:   target,
		Options:  options,
		Commands: commands,
	}
}

//NewExtractRequestFromURL creates a new request from URL
func NewExtractRequestFromURL(URL string) (*ExtractRequest, error) {
	var resource = url.NewResource(URL)
	var result = &ExtractRequest{}
	return result, resource.Decode(result)
}

//Command represents a command expression:  [when criteria ?] command
type Command string

//String returns command string
func (c Command) String() string {
	return string(c)
}

//WhenAndCommand extract when criteria and command
func (c Command) WhenAndCommand() (string, string) {
	var expr = c.String()
	var when, command string
	var variableIndex = strings.Index(expr, "$")
	var criteriaEndIndex = strings.LastIndex(expr, "?")
	if variableIndex == -1 || variableIndex > criteriaEndIndex {
		return when, expr
	}
	when = string(expr[:criteriaEndIndex])
	command = strings.TrimSpace(string(expr[criteriaEndIndex+1:]))
	return when, command
}

//RunRequest represents a simple command
type RunRequest struct {
	Target *url.Resource `required:"true" description:"host where command runs" ` //execution target - destination where to run a command.
	*Options
	Commands []Command `required:"true" description:"command list" ` //list of commands to run
}

//Init initialises request
func (r *RunRequest) Init() error {
	if r.Options == nil {
		r.Options = DefaultOptions()
	}
	return nil
}

//Validate validates managed command request
func (r *RunRequest) Validate() error {
	if r.Target == nil {
		return fmt.Errorf("target was empty")
	}
	if r.Commands == nil {
		return fmt.Errorf("commands were empty")
	}
	return nil
}

//AsExtractRequest returns ExtractRequest for this requests
func (r *RunRequest) AsExtractRequest() *ExtractRequest {
	var request = &ExtractRequest{
		Options:  r.Options,
		Target:   r.Target,
		Commands: make([]*ExtractCommand, 0),
	}
	if len(r.Errors) == 0 {
		r.Errors = []string{}
	}
	var commandErrors = append(CommandErrors, r.Errors...)
	for _, command := range r.Commands {
		when, runCommand := command.WhenAndCommand()
		request.Commands = append(request.Commands,
			&ExtractCommand{
				When:    when,
				Command: runCommand,
				Errors:  commandErrors,
			},
		)
	}
	return request
}

//NewRunRequest creates a new request
func NewRunRequest(target *url.Resource, superUser bool, commands ...string) *RunRequest {
	requestCommands := make([]Command, 0)
	for _, command := range commands {
		requestCommands = append(requestCommands, Command(command))
	}
	result := &RunRequest{
		Target:   target,
		Options:  DefaultOptions(),
		Commands: requestCommands,
	}

	result.SuperUser = superUser
	return result
}

//NewExtractRequestFromURL creates a new request from URL
func NewRunRequestFromURL(URL string) (*RunRequest, error) {
	var resource = url.NewResource(URL)
	var result = &RunRequest{}
	return result, resource.Decode(result)
}

//Log represents an executed command with Stdin, Stdout or Error
type Log struct {
	Stdin  string
	Stdout string
	Error  string
}

//RunResponse represents a command response with logged commands.
type RunResponse struct {
	Session string
	Cmd     []*Log
	Output  string
	Data    data.Map
	Error   string
}

//OpenSessionRequest represents an open session request.
type OpenSessionRequest struct {
	Target        *url.Resource      //Session is created from target host (servername, port)
	Config        *ssh.SessionConfig //ssh configuration
	SystemPaths   []string           //system path that are applied to the ssh session
	Env           map[string]string
	Transient     bool        //if this flag is true, caller is responsible for closing session, othewise session is closed as context is closed
	Basedir       string      //capture all ssh service command in supplied dir (for unit test only)
	ReplayService ssh.Service //use Ssh ReplayService instead of actual SSH service (for unit test only)
}

//Validate checks if request is valid
func (r *OpenSessionRequest) Validate() error {
	if r.Target == nil {
		return errors.New("target was empty")
	}
	return nil
}

//NewOpenSessionRequest creates a new session if transient flag is true, caller is responsible for closing session, otherwise session is closed as context is closed
func NewOpenSessionRequest(target *url.Resource, systemPaths []string, env map[string]string, transient bool, basedir string) *OpenSessionRequest {
	if len(systemPaths) == 0 {
		systemPaths = []string{}
	}
	if len(env) == 0 {
		env = make(map[string]string)
	}
	return &OpenSessionRequest{
		Target:      target,
		SystemPaths: systemPaths,
		Env:         env,
		Transient:   transient,
		Basedir:     basedir,
	}
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

//Add appends provided log into commands slice.
func (i *RunResponse) Add(log *Log) {
	if len(i.Cmd) == 0 {
		i.Cmd = make([]*Log, 0)
	}
	i.Cmd = append(i.Cmd, log)
}

//Stdout returns stdout for provided index, or all concatenated otherwise
func (i *RunResponse) Stdout(indexes ...int) string {
	if len(indexes) == 0 {
		var result = make([]string, len(i.Cmd))
		for j, stream := range i.Cmd {
			result[j] = stream.Stdout
		}
		return strings.Join(result, "\r\n")
	}
	var result = make([]string, len(indexes))
	for _, index := range indexes {
		if index < len(i.Cmd) {
			result = append(result, i.Cmd[index].Stdout)
		}
	}
	return strings.Join(result, "\r\n")
}

//NewRunResponse creates a new RunResponse
func NewRunResponse(session string) *RunResponse {
	return &RunResponse{
		Session: session,
		Cmd:     make([]*Log, 0),
		Data:    make(map[string]interface{}),
	}
}

//NewCommandLog creates a new command log
func NewCommandLog(stdin, stdout string, err error) *Log {
	result := &Log{
		Stdin: stdin,
	}
	if err != nil {
		result.Error = fmt.Sprintf("%v", err)
	} else {
		result.Stdout = stdout
	}
	return result
}
