package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/ssh"
	"strings"
)

const commandNotFound = "command not found"
const ExecServiceId = "exec"

type OpenSession struct {
	Target      *Resource //target name creates a named session
	Config      *ssh.SessionConfig
	SystemPaths []string
	Transient   bool
}

type CloseSession struct {
	Name string
}

type ExecutionOptions struct {
	SystemPaths []string //path that will be added to the system paths
	Terminators []string //fragment that helps identify that command has been completed - the best is to leave it empty, which is the detected bash prompt
	TimeoutMs   int      //time after command was issued for waiting for command output if expect fragment were not matched.
	Directory   string
	Env         map[string]string
}

func NewExecutionOptions() *ExecutionOptions {
	return &ExecutionOptions{
		SystemPaths: make([]string, 0),
		Terminators: make([]string, 0),
		Env:         make(map[string]string),
	}
}

type ManagedCommand struct {
	Options    *ExecutionOptions
	Executions []*Execution
}

type Execution struct {
	MatchOutput string //only run this execution is output from a previous command is matched
	Command     string
	Extraction  DataExtractions
	Error       []string //fragments that will terminate execution with error if matched with standard output
	Success     []string //if specified absence of all of the these fragment will terminate execution with error.
}

type CommandRequest struct {
	Target        *Resource
	MangedCommand *ManagedCommand
}

type SuperUserCommandRequest struct {
	Target        *Resource
	MangedCommand *ManagedCommand
}

func (r *SuperUserCommandRequest) AsCommandRequest(context *Context) (*CommandRequest, error) {
	target, err := context.ExpandResource(r.Target)
	if err != nil {
		return nil, err
	}
	var result = &CommandRequest{
		Target: target,
		MangedCommand: &ManagedCommand{
			Executions: make([]*Execution, 0),
		},
	}
	var executionOptions = &ExecutionOptions{
		Terminators: []string{"Password"},
	}
	if r.MangedCommand.Options != nil {
		executionOptions.Terminators = append(executionOptions.Terminators, r.MangedCommand.Options.Terminators...)
		executionOptions.TimeoutMs = r.MangedCommand.Options.TimeoutMs
		executionOptions.Directory = r.MangedCommand.Options.Directory
		executionOptions.SystemPaths = r.MangedCommand.Options.SystemPaths
	}

	var errors = make([]string, 0)
	var extractions = make([]*DataExtraction, 0)
	for _, execution := range r.MangedCommand.Executions {
		if execution.Command == "" {
			continue
		}
		newExecution := &Execution{
			Command:     "sudo " + execution.Command,
			Error:       execution.Error,
			Extraction:  execution.Extraction,
			Success:     execution.Success,
			MatchOutput: execution.MatchOutput,
		}
		if len(execution.Error) > 0 {
			errors = append(errors, execution.Error...)
		}
		if len(execution.Extraction) > 0 {
			extractions = append(extractions, execution.Extraction...)
		}
		result.MangedCommand.Executions = append(result.MangedCommand.Executions, newExecution)
	}

	_, password, err := target.LoadCredential()
	execution := &Execution{
		MatchOutput: "Password",
		Command:     password,
		Error:       []string{"Password"},
		Extraction:  extractions,
	}
	execution.Error = append(execution.Error, errors...)
	result.MangedCommand.Executions = append(result.MangedCommand.Executions, execution)
	return result, nil
}

func NewCommandRequest(target *Resource, execution *ManagedCommand) *CommandRequest {
	return &CommandRequest{
		Target:        target,
		MangedCommand: execution,
	}
}

type execService struct {
	*AbstractService
}

type ClientSession struct {
	*ssh.MultiCommandSession
	Connection      *ssh.Client
	OperatingSystem *OperatingSystem
}

type ClientSessions map[string]*ClientSession

func (s *ClientSessions) Has(name string) bool {
	_, has := (*s)[name]
	return has
}

type CommandResult struct {
	Commands  []string
	Stdout    []string
	Extracted map[string]string
}

var clientSessionKey = (*ClientSessions)(nil)

func (s *execService) openSession(context *Context, request *OpenSession) (interface{}, error) {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	if !(target.ParsedURL.Scheme == "ssh" || target.ParsedURL.Scheme == "scp") {
		return nil, fmt.Errorf("Failed to open session: invalid schema: %v", target.ParsedURL.Scheme)
	}
	clientSessions := context.Sessions()
	var session = target.Session()
	if clientSessions.Has(session) {
		clientSession := clientSessions[session]
		_, err = clientSession.Run(fmt.Sprintf("cd %v", target.ParsedURL.Path), 0)
		if err != nil {
			return nil, err
		}
		return clientSessions[session], nil
	}
	clientSession := &ClientSession{}

	manager, err := context.ServiceManager()
	if err != nil {
		return nil, err
	}

	var file = ""
	var authConfig = &ssh.AuthConfig{}
	targetCredential := target.Credential
	if targetCredential != "" {
		targetCredential = context.Expand(targetCredential)
		file, err = manager.CredentialFile(targetCredential)
		if err != nil {
			return nil, err
		}
		authConfig, err = ssh.NewAuthConfigFromURL(fmt.Sprintf("file://%v", file))
		if err != nil {
			return nil, err
		}
	}

	port := toolbox.AsInt(target.ParsedURL.Port())
	if port == 0 {
		port = 22
	}
	clientSession.Connection, err = ssh.NewClient(target.ParsedURL.Hostname(), port, authConfig)
	if err != nil {
		return nil, err
	}

	if !request.Transient {
		context.Deffer(func() {
			clientSession.Connection.Close()
		})
	}

	clientSession.MultiCommandSession, err = clientSession.Connection.OpenMultiCommandSession(request.Config)
	if err != nil {
		return nil, err
	}
	if !request.Transient {
		context.Deffer(func() {
			clientSession.MultiCommandSession.Close()
		})
	}

	_, err = clientSession.Run(fmt.Sprintf("cd %v", target.ParsedURL.Path), 0)
	if err != nil {
		return nil, err
	}

	clientSessions[session] = clientSession
	clientSession.OperatingSystem, err = s.detectOperatingSystem(clientSession)
	if err != nil {
		return nil, err
	}
	return clientSession, nil
}

func (s *execService) applyCommandOptions(context *Context, options *ExecutionOptions, sesssion *ClientSession) error {

	operatingSystem := sesssion.OperatingSystem
	if options == nil {
		return nil
	}

	var timeoutMs = options.TimeoutMs
	if len(options.SystemPaths) > 0 {
		operatingSystem.Path.Push(options.SystemPaths...)
	}

	for k, v := range options.Env {
		_, err := sesssion.Run(fmt.Sprintf("export %v='%v'", k, v), timeoutMs)
		if err != nil {
			return err
		}
	}
	if options.Directory != "" {
		directory := context.Expand(options.Directory)
		_, err := sesssion.Run(fmt.Sprintf("cd %v", directory), timeoutMs)
		if err != nil {
			return err
		}
	}
	return nil
}

func match(stdout string, candidates ...string) string {
	if len(candidates) == 0 {
		return ""
	}
	for _, candidate := range candidates {
		if strings.Contains(stdout, candidate) {
			return candidate
		}
	}
	return ""
}

func (s *execService) executeCommand(context *Context, session *ClientSession, execution *Execution, options *ExecutionOptions, result *CommandResult, request *CommandRequest) error {
	command := context.Expand(execution.Command)
	result.Commands = append(result.Commands, command)

	var terminators = append([]string{}, options.Terminators...)
	terminators = append(terminators, "$ ")
	terminators = append(terminators, session.ShellPrompt)

	superUserPrompt := string(strings.Replace(session.ShellPrompt, "$", "#", 1))
	if strings.Contains(superUserPrompt, "bash") {
		superUserPrompt = string(superUserPrompt[2:])
	}
	terminators = append(terminators, superUserPrompt)

	terminators = append(terminators, execution.Error...)

	output, err := session.Run(command, options.TimeoutMs, terminators...)
	if err != nil {
		return err
	}
	stdout := string(output)
	result.Stdout = append(result.Stdout, stdout)
	errorMatch := match(stdout, execution.Error...)
	if errorMatch != "" {
		return fmt.Errorf("Encounter error fragment: (%v) execution (%v); ouput: (%v), %v", errorMatch, execution.Command, stdout, options.Directory)
	}
	if len(execution.Success) > 0 {
		sucessMatch := match(stdout, execution.Success...)
		if sucessMatch == "" {
			return fmt.Errorf("Fail to match any fragment: (%v) execution (%v); ouput: (%v), %v", strings.Join(execution.Success, ","), execution.Command, stdout, options.Directory)
		}
	}
	err = execution.Extraction.Extract(context, result.Extracted, strings.Split(stdout, "\r\n")...)
	if err != nil {
		return err
	}
	if len(stdout) > 0 {
		for _, execution := range request.MangedCommand.Executions {
			if execution.MatchOutput != "" && strings.Contains(stdout, execution.MatchOutput) {
				return s.executeCommand(context, session, execution, options, result, request)
			}
		}
	}
	return nil
}

func (s *execService) runCommands(context *Context, request *CommandRequest) (*CommandResult, error) {
	clientSessions := context.Sessions()
	result := &CommandResult{
		Commands:  make([]string, 0),
		Stdout:    make([]string, 0),
		Extracted: make(map[string]string),
	}

	var target, err = context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	s.openSession(context, &OpenSession{Target: target})
	var sessionName = target.Session()
	session, has := clientSessions[sessionName]
	if !has {
		return nil, fmt.Errorf("Failed to lookup sessionName: %v", sessionName)
	}

	var options = request.MangedCommand.Options
	if options == nil {
		options = NewExecutionOptions()
	}
	err = s.applyCommandOptions(context, options, session)
	if err != nil {
		return nil, err
	}

	operatingSystem := session.OperatingSystem
	var systemPath = fmt.Sprintf("export PATH=\"%v\"", operatingSystem.Path.EnvValue())
	_, err = session.Run(systemPath, 0)
	if err != nil {
		return nil, err
	}

	for _, execution := range request.MangedCommand.Executions {

		if execution.MatchOutput != "" {
			continue
		}
		err = s.executeCommand(context, session, execution, options, result, request)
		if err != nil {
			return nil, err
		}

	}
	return result, nil
}

func (s *execService) closeSession(context *Context, closeSession *CloseSession) (interface{}, error) {
	clientSessions := context.Sessions()
	if session, has := clientSessions[closeSession.Name]; has {
		session.Close()
	}
	if connection, has := clientSessions[closeSession.Name]; has {
		connection.Close()
	}
	return nil, nil
}

func (s *execService) Run(context *Context, request interface{}) *Response {
	var response = &Response{
		Status: "ok",
	}
	switch castedRequest := request.(type) {
	case *OpenSession:
		response.Response, response.Error = s.openSession(context, castedRequest)
	case *CommandRequest:
		response.Response, response.Error = s.runCommands(context, castedRequest)
	case *SuperUserCommandRequest:
		commandRequest, err := castedRequest.AsCommandRequest(context)
		if err == nil {
			response.Response, response.Error = s.runCommands(context, commandRequest)
		} else {
			response.Error = err
		}

	case *CloseSession:
		response.Response, response.Error = s.closeSession(context, castedRequest)

	default:
		response.Error = fmt.Errorf("Unsupported request type: %T", request)
	}

	if response.Error != nil {
		response.Status = "error"
	}
	return response
}

func (s *execService) NewRequest(name string) (interface{}, error) {
	switch name {
	case "open":
		return &OpenSession{}, nil
	case "command":
		return &CommandRequest{}, nil
	case "close":
		return &CloseSession{}, nil

	}
	return nil, fmt.Errorf("Unsupported request: %v", name)
}

func (s *execService) detectOperatingSystem(session *ClientSession) (*OperatingSystem, error) {
	operatingSystem := &OperatingSystem{
		Path: &SystemPath{
			SystemPath: make([]string, 0),
			Path:       make([]string, 0),
			index:      make(map[string]bool),
		},
	}
	var releaseCommands = []string{
		"sw_vers",
		"lsb_release -a",
	}
	var stdout string
	for _, command := range releaseCommands {
		output, err := session.Run(command, 0)
		if err != nil {
			return nil, err
		}
		stdout = string(output)
		if !strings.Contains(stdout, commandNotFound) {
			break
		}
	}
	lines := strings.Split(stdout, "\r\n")
	for _, line := range lines {
		pair := strings.Split(line, ":")
		if len(pair) != 2 {
			continue
		}
		var key = strings.Replace(strings.ToLower(pair[0]), " ", "", len(pair[0]))
		var val = strings.Replace(strings.Trim(pair[1], " \t\r"), " ", "", len(line))
		switch key {
		case "productname", "distributorid":
			operatingSystem.Name = val
		case "productversion", "release":
			operatingSystem.Version = val
		}
	}

	output, err := session.Run("echo $PATH", 0)
	if err != nil {
		return nil, err
	}
	stdOut := string(output)
	operatingSystem.Path.SystemPath = strings.Split(stdOut, ":")
	return operatingSystem, nil
}

func NewExecService() Service {
	var result = &execService{
		AbstractService: NewAbstractService(ExecServiceId),
	}
	result.AbstractService.Service = result
	return result
}
