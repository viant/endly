package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/ssh"
	"strings"
)

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
	Secure      string
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
		Secure:      password,
		MatchOutput: "Password",
		Command:     "****",
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

func NewSimpleCommandRequest(target *Resource, commands ...string) *CommandRequest {
	var result = &CommandRequest{
		Target: target,
		MangedCommand: &ManagedCommand{
			Executions: make([]*Execution, 0),
		},
	}
	for _, command := range commands {
		result.MangedCommand.Executions = append(result.MangedCommand.Executions, &Execution{
			Command: command,
		})
	}
	return result
}

type CommandStream struct {
	Stdin  string
	Stdout string
	Error  string
}

func NewCommandStream(stdin, stdout string, err error) *CommandStream {
	result := &CommandStream{
		Stdin: stdin,
	}
	if err != nil {
		result.Error = fmt.Sprintf("%v", err)
	} else {
		result.Stdout = stdout
	}
	return result
}

type CommandInfo struct {
	Session   string
	Commands  []*CommandStream
	Extracted map[string]string
	Error     string
}

func (i *CommandInfo) Add(stream *CommandStream) {
	if len(i.Commands) == 0 {
		i.Commands = make([]*CommandStream, 0)
	}
	i.Commands = append(i.Commands, stream)
}

func (i *CommandInfo) Stdout(indexes ...int) string {
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

func NewCommandInfo(session string) *CommandInfo {
	return &CommandInfo{
		Session:   session,
		Commands:  make([]*CommandStream, 0),
		Extracted: make(map[string]string),
	}
}

type execService struct {
	*AbstractService
}

type ClientSession struct {
	name string
	*ssh.MultiCommandSession
	Connection      *ssh.Client
	OperatingSystem *OperatingSystem
	envVariables    map[string]string
	path            string
}

func NewClientSession(name string, connection *ssh.Client) (*ClientSession, error) {

	return &ClientSession{
		name:         name,
		Connection:   connection,
		envVariables: make(map[string]string),
	}, nil
}

type ClientSessions map[string]*ClientSession

func (s *ClientSessions) Has(name string) bool {
	_, has := (*s)[name]
	return has
}

var clientSessionKey = (*ClientSessions)(nil)

func (s *execService) openSession(context *Context, request *OpenSession) (*ClientSession, error) {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}

	if !(target.ParsedURL.Scheme == "ssh" || target.ParsedURL.Scheme == "scp" || target.ParsedURL.Scheme == "file") {
		return nil, fmt.Errorf("Failed to open sessionName: invalid schema: %v in url: %v", target.ParsedURL.Scheme, target.URL)
	}
	sessions := context.Sessions()

	var sessionName = target.Session()
	if sessions.Has(sessionName) {
		session := sessions[sessionName]
		err = s.changeDirectory(context, session, nil, target.ParsedURL.Path)
		return sessions[sessionName], err
	}
	var authConfig = &ssh.AuthConfig{}
	_ = LoadCredential(target.CredentialFile, authConfig)
	if err != nil {
		return nil, err
	}
	hostname, port := getHostAndSSHPort(target)
	connection, err := ssh.NewClient(hostname, port, authConfig)
	if err != nil {
		return nil, err
	}
	session, err := NewClientSession(sessionName, connection)
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

func getHostAndSSHPort(target *Resource) (string, int) {
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

func (s *execService) setEnvVariable(context *Context, session *ClientSession, info *CommandInfo, name, value string) error {
	if val, has := session.envVariables[name]; has {
		if value == val {
			return nil
		}
		session.envVariables[name] = value
	}
	return s.rumCommandTemplate(context, session, info, "export %v='%v'", name, value)
}

func (s *execService) changeDirectory(context *Context, session *ClientSession, commandInfo *CommandInfo, directory string) error {
	if session.path == directory {
		return nil
	}
	session.path = directory
	return s.rumCommandTemplate(context, session, commandInfo, "cd %v", directory)
}

func (s *execService) rumCommandTemplate(context *Context, session *ClientSession, info *CommandInfo, commandTemplate string, arguments ...interface{}) error {
	if info == nil {
		info = NewCommandInfo(session.name)
		context.SessionInfo().Log(info)
	}
	command := fmt.Sprintf(commandTemplate, arguments...)
	output, err := session.Run(command, 0)
	info.Add(NewCommandStream(command, output, err))
	if err != nil {
		return err
	}
	return nil
}

func (s *execService) applyCommandOptions(context *Context, options *ExecutionOptions, session *ClientSession, info *CommandInfo) error {

	operatingSystem := session.OperatingSystem
	if options == nil {
		return nil
	}

	if len(options.SystemPaths) > 0 {
		operatingSystem.Path.Push(options.SystemPaths...)
	}
	for k, v := range options.Env {
		err := s.setEnvVariable(context, session, info, k, v)
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

func (s *execService) executeCommand(context *Context, session *ClientSession, execution *Execution, options *ExecutionOptions, commandInfo *CommandInfo, request *CommandRequest) error {
	command := context.Expand(execution.Command)
	terminators := getTerminators(options, session, execution)

	var cmd = command
	if execution.Secure != "" {
		cmd = strings.Replace(command, "****", execution.Secure, 1)
	}

	stdout, err := session.Run(cmd, options.TimeoutMs, terminators...)

	//fmt.Printf("IN: %v\nOUT:%v\n", cmd, stdout)
	commandInfo.Add(NewCommandStream(command, stdout, err))
	if err != nil {
		return err
	}
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
		for _, execution := range request.MangedCommand.Executions {
			if execution.MatchOutput != "" && strings.Contains(stdout, execution.MatchOutput) {
				return s.executeCommand(context, session, execution, options, commandInfo, request)
			}
		}
	}
	return nil
}
func getTerminators(options *ExecutionOptions, session *ClientSession, execution *Execution) []string {
	var terminators = append([]string{}, options.Terminators...)
	terminators = append(terminators, "$ ")
	terminators = append(terminators, session.ShellPrompt)
	superUserPrompt := string(strings.Replace(session.ShellPrompt, "$", "#", 1))
	if strings.Contains(superUserPrompt, "bash") {
		superUserPrompt = string(superUserPrompt[2:])
	}
	terminators = append(terminators, superUserPrompt)
	terminators = append(terminators, execution.Error...)
	return terminators
}

func (s *execService) runCommands(context *Context, request *CommandRequest) (*CommandInfo, error) {
	//clientSessions := context.Sessions()
	var target, err = context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	session, err := s.openSession(context, &OpenSession{Target: target})
	if err != nil {
		return nil, err
	}

	//session, has := clientSessions[sessionName]
	//if !has {
	//	return nil, fmt.Errorf("Failed to lookup sessionName: %v", sessionName)
	//}

	var options = request.MangedCommand.Options
	if options == nil {
		options = NewExecutionOptions()
	}
	info := NewCommandInfo(session.name)
	context.SessionInfo().Log(info)
	err = s.applyCommandOptions(context, options, session, info)
	if err != nil {
		return nil, err
	}

	operatingSystem := session.OperatingSystem
	if session.path != operatingSystem.Path.EnvValue() {
		session.path = operatingSystem.Path.EnvValue()
		err := s.setEnvVariable(context, session, info, "PATH", session.path)
		if err != nil {
			return nil, err
		}
	}
	info = NewCommandInfo(session.name)
	context.SessionInfo().Log(info)
	for _, execution := range request.MangedCommand.Executions {
		if execution.MatchOutput != "" {
			continue
		}
		if strings.HasPrefix(execution.Command, "cd ") {
			session.path = "" //reset path
		}
		if strings.HasPrefix(execution.Command, "export ") {
			session.envVariables = make(map[string]string) //reset env variables
		}
		err = s.executeCommand(context, session, execution, options, info, request)
		if err != nil {
			return nil, err
		}

	}
	return info, nil
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

func (s *execService) Run(context *Context, request interface{}) *ServiceResponse {

	var response = &ServiceResponse{
		Status: "ok",
	}
	var err error
	switch actualRequest := request.(type) {
	case *OpenSession:
		response.Response, err = s.openSession(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to open session: %v, %v", actualRequest.Target, err)
		}
	case *CommandRequest:
		response.Response, err = s.runCommands(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to run command: %v, %v", actualRequest.MangedCommand, err)
		}
	case *SuperUserCommandRequest:
		commandRequest, err := actualRequest.AsCommandRequest(context)
		if err == nil {
			response.Response, err = s.runCommands(context, commandRequest)
		}
		if err != nil {
			response.Error = fmt.Sprintf("%v", err)
		}

	case *CloseSession:
		response.Response, err = s.closeSession(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to close session: %v, %v", actualRequest.Name, err)
		}

	default:
		response.Error = fmt.Sprintf("Unsupported request type: %T", request)
	}

	if response.Error != "" {
		response.Status = "error"
	}
	return response
}

func (s *execService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "open":
		return &OpenSession{}, nil
	case "command":
		return &CommandRequest{}, nil
	case "close":
		return &CloseSession{}, nil

	}
	return nil, fmt.Errorf("Unsupported action: %v", action)
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
		if !CheckCommandNotFound(stdout) {
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
