package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/ssh"
	"github.com/viant/toolbox/url"
	"strings"
	"path"
	"github.com/lunixbochs/vtclean"
)

//SystemExecServiceID represent system executor service id
const SystemExecServiceID = "exec"

//ExecutionStartEvent represents an execution event start
type ExecutionStartEvent struct {
	SessionID string
	Stdin     string
}

//ExecutionEndEvent represents an execution event end
type ExecutionEndEvent struct {
	SessionID string
	Stdout    string
	Error     string
}

type execService struct {
	*AbstractService
}

func (s *execService) open(context *Context, request *OpenSessionRequest) (*OpenSessionResponse, error) {
	var clientSession, err = s.openSession(context, request)
	if err != nil {
		return nil, err
	}
	return &OpenSessionResponse{
		SessionID: clientSession.ID,
	}, nil
}

func (s *execService) openSession(context *Context, request *OpenSessionRequest) (*SystemTerminalSession, error) {
	s.Mutex().Lock()
	defer s.Mutex().Unlock()
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	if !(target.ParsedURL.Scheme == "ssh" || target.ParsedURL.Scheme == "scp" || target.ParsedURL.Scheme == "file") {
		return nil, fmt.Errorf("Failed to open sessionName: invalid schema: %v in url: %v", target.ParsedURL.Scheme, target.URL)
	}
	sessions := context.TerminalSessions()

	var sessionName = target.Host()
	if sessions.Has(sessionName) {
		session := sessions[sessionName]
		err = s.changeDirectory(context, session, nil, target.ParsedURL.Path)
		return sessions[sessionName], err
	}
	var authConfig = &cred.Config{}

	if target.Credential != "" {
		err = authConfig.Load(target.Credential)
		if err != nil {
			return nil, err
		}
	}
	hostname, port := getHostAndSSHPort(target)
	connection, err := ssh.NewService(hostname, port, authConfig)
	if err != nil {
		return nil, err
	}
	session, err := NewSystemTerminalSession(sessionName, connection)
	if err != nil {
		return nil, err
	}

	if !request.Transient {
		context.Deffer(func() {
			connection.Close()
		})
	}

	session.MultiCommandSession, err = session.Connection.OpenMultiCommandSession(request.Config)
	if err != nil {
		return nil, err
	}
	if !request.Transient {
		context.Deffer(func() {
			session.MultiCommandSession.Close()
		})
	}
	err = s.changeDirectory(context, session, nil, target.ParsedURL.Path)
	if err != nil {
		return nil, err
	}
	sessions[sessionName] = session
	session.OperatingSystem, err = s.detectOperatingSystem(session)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func getHostAndSSHPort(target *url.Resource) (string, int) {
	port := toolbox.AsInt(target.ParsedURL.Port())
	if port == 0 {
		port = 22
	}
	hostname := target.ParsedURL.Hostname()
	if hostname == "" {
		hostname = "127.0.0.1"
	}
	return hostname, port
}

func (s *execService) setEnvVariable(context *Context, session *SystemTerminalSession, name, value string) error {
	value = context.Expand(value)
	if val, has := session.envVariables[name]; has {
		if value == val {
			return nil
		}
		session.envVariables[name] = value
	}
	return s.rumCommandTemplate(context, session, "export %v='%v'", name, value)
}

func (s *execService) changeDirectory(context *Context, session *SystemTerminalSession, commandInfo *CommandResponse, directory string) error {
	parent, name := path.Split(directory)
	if path.Ext(name) != "" {
		directory = parent
	}
	if session.path == directory {
		return nil
	}
	session.path = directory
	return s.rumCommandTemplate(context, session, "cd %v", directory)
}

func (s *execService) rumCommandTemplate(context *Context, session *SystemTerminalSession, commandTemplate string, arguments ...interface{}) error {
	command := fmt.Sprintf(commandTemplate, arguments...)
	var executionStartEvent = &ExecutionStartEvent{SessionID: session.ID, Stdin: command}
	startEvent := s.Begin(context, executionStartEvent, Pairs("value", executionStartEvent), Info)
	stdout, err := session.Run(command, 1000)
	var executionEndEvent = &ExecutionEndEvent{
		SessionID: session.ID,
		Stdout:    stdout,
	}
	s.End(context)(startEvent, Pairs("value", executionEndEvent))
	if err != nil {
		executionEndEvent.Error = fmt.Sprintf("%v", err)
	}
	if err != nil {
		return err
	}
	return nil
}

func (s *execService) applyCommandOptions(context *Context, options *ExecutionOptions, session *SystemTerminalSession, info *CommandResponse) error {

	operatingSystem := session.OperatingSystem
	if options == nil {
		return nil
	}

	if len(options.SystemPaths) > 0 {
		operatingSystem.Path.Push(options.SystemPaths...)
	}
	for k, v := range options.Env {
		err := s.setEnvVariable(context, session, k, v)
		if err != nil {
			return err
		}
	}
	if options.Directory != "" {
		directory := context.Expand(options.Directory)
		err := s.changeDirectory(context, session, info, directory)
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

func (s *execService) executeCommand(context *Context, session *SystemTerminalSession, execution *Execution, options *ExecutionOptions, commandInfo *CommandResponse, request *ManagedCommandRequest) error {
	command := context.Expand(execution.Command)
	terminators := getTerminators(options, session, execution)

	var cmd = command
	if execution.Secure != "" {
		cmd = strings.Replace(command, "****", execution.Secure, 1)
	}


	var executionStartEvent = &ExecutionStartEvent{SessionID: session.ID, Stdin: command}
	startEvent := s.Begin(context, executionStartEvent, Pairs("value", executionStartEvent), Info)
	stdout, err := session.Run(cmd, options.TimeoutMs, terminators...)
	var executionEndEvent = &ExecutionEndEvent{
		SessionID: session.ID,
		Stdout:    stdout,
	}
	if err != nil {
		executionEndEvent.Error = fmt.Sprintf("%v", err)
	}
	s.End(context)(startEvent, Pairs("value", executionEndEvent))
	commandInfo.Add(NewCommandLog(command, stdout, err))
	if err != nil {
		return err
	}

	stdout = vtclean.Clean(stdout, false)
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
	err = execution.Extraction.Extract(context, commandInfo.Extracted, strings.Split(stdout, "\r\n")...)
	if err != nil {
		return err
	}

	if len(stdout) > 0 {
		for _, execution := range request.ManagedCommand.Executions {
			if execution.MatchOutput != "" && strings.Contains(stdout, execution.MatchOutput) {
				return s.executeCommand(context, session, execution, options, commandInfo, request)
			}
		}
	}
	return nil
}
func getTerminators(options *ExecutionOptions, session *SystemTerminalSession, execution *Execution) []string {
	var terminators = append([]string{}, options.Terminators...)
	terminators = append(terminators, "$ ")
	superUserPrompt := string(strings.Replace(session.ShellPrompt(), "$", "#", 1))
	if strings.Contains(superUserPrompt, "bash") {
		superUserPrompt = string(superUserPrompt[2:])
	}
	terminators = append(terminators, superUserPrompt)
	terminators = append(terminators, execution.Error...)
	return terminators
}

func (s *execService) runCommands(context *Context, request *ManagedCommandRequest) (*CommandResponse, error) {

	err := request.Validate()
	if err != nil {
		return nil, err
	}
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	session, err := s.openSession(context, &OpenSessionRequest{Target: target})
	if err != nil {
		return nil, err
	}

	var options = request.ManagedCommand.Options
	if options == nil {
		options = NewExecutionOptions()
	}
	response := NewCommandResponse(session.ID)
	err = s.applyCommandOptions(context, options, session, response)

	if err != nil {
		return nil, err
	}

	operatingSystem := session.OperatingSystem
	if session.path != operatingSystem.Path.EnvValue() {
		session.path = operatingSystem.Path.EnvValue()
		err := s.setEnvVariable(context, session, "PATH", session.path)
		if err != nil {
			return nil, err
		}
	}
	response = NewCommandResponse(session.ID)
	for _, execution := range request.ManagedCommand.Executions {

		if execution.MatchOutput != "" {
			continue
		}
		if strings.HasPrefix(execution.Command, "cd ") {
			session.path = "" //reset path
		}
		if strings.HasPrefix(execution.Command, "export ") {
			session.envVariables = make(map[string]string) //reset env variables
		}
		err = s.executeCommand(context, session, execution, options, response, request)
		if err != nil {
			return nil, err
		}

	}
	return response, nil
}

func (s *execService) closeSession(context *Context, request *CloseSessionRequest) (*CloseSessionResponse, error) {
	clientSessions := context.TerminalSessions()
	if session, has := clientSessions[request.SessionID]; has {
		session.Close()
	}
	if connection, has := clientSessions[request.SessionID]; has {
		connection.Close()
	}
	return &CloseSessionResponse{
		SessionID: request.SessionID,
	}, nil
}

//Run runs action for passed in request.
func (s *execService) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok"}
	defer s.End(context)(startEvent, Pairs("response", response))
	var err error
	switch actualRequest := request.(type) {
	case *CommandRequest:
		var mangedCommandRequest = actualRequest.AsManagedCommandRequest()
		if actualRequest.SuperUser {
			superCommandRequest := superUserCommandRequest{
				Target:        actualRequest.Target,
				MangedCommand: mangedCommandRequest.ManagedCommand,
			}
			mangedCommandRequest, err = superCommandRequest.AsCommandRequest(context)
		}
		if err == nil {
			response.Response, err = s.runCommands(context, mangedCommandRequest)
		}
		if err != nil {
			response.Error = fmt.Sprintf("Failed to run command: %v, %v", actualRequest, err)
		}

	case *OpenSessionRequest:
		response.Response, err = s.open(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to open session: %v, %v", actualRequest.Target, err)
		}
	case *ManagedCommandRequest:
		response.Response, err = s.runCommands(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to run command: %v, %v", actualRequest.ManagedCommand, err)
		}
	case *superUserCommandRequest:
		commandRequest, err := actualRequest.AsCommandRequest(context)
		if err == nil {
			response.Response, err = s.runCommands(context, commandRequest)
		}
		if err != nil {
			response.Error = fmt.Sprintf("%v", err)
		}

	case *CloseSessionRequest:
		response.Response, err = s.closeSession(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to close session: %v, %v", actualRequest.SessionID, err)
		}

	default:
		response.Error = fmt.Sprintf("Unsupported request type: %T", request)
	}

	if response.Error != "" {
		response.Status = "error"
	}
	return response
}

//NewRequest creates a new request for passed in action, the following is supported: open,close,command,managedCommand
func (s *execService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "open":
		return &OpenSessionRequest{}, nil
	case "managedCommand":
		return &ManagedCommandRequest{}, nil
	case "command":
		return &CommandRequest{}, nil
	case "close":
		return &CloseSessionRequest{}, nil

	}
	return nil, fmt.Errorf("Unsupported action: %v", action)
}

func (s *execService) detectOperatingSystem(session *SystemTerminalSession) (*OperatingSystem, error) {
	operatingSystem := &OperatingSystem{
		Path: &SystemPath{
			SystemPath: make([]string, 0),
			Path:       make([]string, 0),
			index:      make(map[string]bool),
		},
	}
	var releaseCommands = []string{
		"lsb_release -a",
	}
	var stdout string
	for _, command := range releaseCommands {
		output, err := session.Run(command, 0)
		if err != nil {
			return nil, err
		}
		if CheckCommandNotFound(string(output)) {
			continue
		}
		stdout += string(output)
	}

	lines := strings.Split(stdout, "\r\n")
	for i := 0; i < len(lines)-1; i++ {
		line := lines[i]
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

	operatingSystem.System = session.System()
	//TODO add os architecure i.e.x64
	output, err := session.Run("echo $PATH", 0)
	if err != nil {
		return nil, err
	}
	stdOut := string(output)
	var newLine = strings.Index(stdOut, "\n")
	if newLine != -1 {
		stdOut = string(stdOut[:newLine])
	}
	operatingSystem.Path.SystemPath = strings.Split(stdOut, ":")
	return operatingSystem, nil
}

//NewExecService creates a new execution service
func NewExecService() Service {
	var result = &execService{
		AbstractService: NewAbstractService(SystemExecServiceID),
	}
	result.AbstractService.Service = result
	return result
}

//superUserCommandRequest represents a super user command,
type superUserCommandRequest struct {
	Target        *url.Resource   //target destination where to run a command.
	MangedCommand *ManagedCommand //managed command
}

//AsCommandRequest returns ManagedCommandRequest
func (r *superUserCommandRequest) AsCommandRequest(context *Context) (*ManagedCommandRequest, error) {
	target, err := context.ExpandResource(r.Target)
	if err != nil {
		return nil, err
	}
	var result = &ManagedCommandRequest{
		Target: target,
		ManagedCommand: &ManagedCommand{
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
	result.ManagedCommand.Options = executionOptions
	var errors = make([]string, 0)
	var extractions = make([]*DataExtraction, 0)
	for _, execution := range r.MangedCommand.Executions {
		if execution.Command == "" {
			continue
		}
		sudo := ""
		if len(execution.Command) > 1 {
			sudo = "sudo "
		}
		newExecution := &Execution{
			Command:     sudo + execution.Command,
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
		result.ManagedCommand.Executions = append(result.ManagedCommand.Executions, newExecution)
	}

	if target.Credential == "" {
		return nil, fmt.Errorf("Can not run as superuser, credential were empty for target: %v", target.URL)
	}
	_, password, err := target.LoadCredential(true)
	execution := &Execution{
		Secure:      password,
		MatchOutput: "Password",
		Command:     "****",
		Error:       []string{"Password"},
		Extraction:  extractions,
	}
	execution.Error = append(execution.Error, errors...)
	result.ManagedCommand.Executions = append(result.ManagedCommand.Executions, execution)
	return result, nil
}
