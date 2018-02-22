package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/ssh"
	"github.com/viant/toolbox/url"
	"path"
	"sort"
	"strings"
	"sync"
)

//ExecServiceID represent system executor service id
const ExecServiceID = "exec"

const sudoCredentialKey = "**sudo**"

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
	mutex       *sync.RWMutex
	credentials map[string]*cred.Config
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

func (s *execService) openSSHService(context *Context, request *OpenSessionRequest) (ssh.Service, error) {
	if request.ReplayService != nil {
		return request.ReplayService, nil
	}
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	var authConfig = &cred.Config{}
	if target.Credential != "" {
		err = authConfig.Load(target.Credential)
		if err != nil {
			return nil, err
		}
	}
	hostname, port := getHostAndSSHPort(target)
	return ssh.NewService(hostname, port, authConfig)
}

func (s *execService) isSupportedScheme(target *url.Resource) bool {
	return target.ParsedURL.Scheme == "ssh" || target.ParsedURL.Scheme == "scp" || target.ParsedURL.Scheme == "file"
}

func (s *execService) initSession(context *Context, target *url.Resource, session *SystemTerminalSession, env map[string]string) error {
	_, _ = s.changeDirectory(context, session, nil, target.ParsedURL.Path)
	for k, v := range env {
		if err := s.setEnvVariable(context, session, k, v); err != nil {
			return err
		}
	}
	return nil
}

func (s *execService) captureCommandIfNeeded(context *Context, replayCommands *ssh.ReplayCommands, sshService ssh.Service) (err error) {
	if replayCommands != nil {
		err = replayCommands.Enable(sshService)
		if err != nil {
			return err
		}
		context.Deffer(func() {
			_ = replayCommands.Store()
		})
	}
	return err
}

func (s *execService) openSession(context *Context, request *OpenSessionRequest) (*SystemTerminalSession, error) {
	s.Mutex().Lock()
	defer s.Mutex().Unlock()
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	if !s.isSupportedScheme(target) {
		return nil, fmt.Errorf("failed to open sessionName: invalid schema: %v in url: %v", target.ParsedURL.Scheme, target.URL)
	}
	sessions := context.TerminalSessions()

	var replayCommands *ssh.ReplayCommands
	if request.CommandsBasedir != "" {
		replayCommands, err = ssh.NewReplayCommands(request.CommandsBasedir)
		if err != nil {
			return nil, err
		}
	}
	var sessionName = target.Host()
	if sessions.Has(sessionName) {
		session := sessions[sessionName]
		err = s.initSession(context, target, session, request.Env)
		if err != nil {
			return nil, err
		}
		return sessions[sessionName], err
	}

	sshService, err := s.openSSHService(context, request)
	if err == nil {
		err = s.captureCommandIfNeeded(context, replayCommands, sshService)
	}
	if err != nil {
		return nil, err
	}
	session, err := NewSystemTerminalSession(sessionName, sshService)
	if err != nil {
		return nil, err
	}
	if !request.Transient {
		context.Deffer(func() {
			_ = sshService.Close()
		})
	}
	session.MultiCommandSession, err = session.Service.OpenMultiCommandSession(request.Config)
	if err != nil {
		return nil, err
	}
	if !request.Transient {
		context.Deffer(func() {
			session.MultiCommandSession.Close()
		})
	}
	err = s.initSession(context, target, session, request.Env)
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
	if target == nil {
		return "", 0
	}
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

func (s *execService) setEnvVariables(context *Context, session *SystemTerminalSession, env map[string]string) error {
	for k, v := range env {
		err := s.setEnvVariable(context, session, k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *execService) setEnvVariable(context *Context, session *SystemTerminalSession, name, newValue string) error {
	newValue = context.Expand(newValue)

	if actual, has := session.envVariables[name]; has {
		if newValue == actual {
			return nil
		}
	}
	session.envVariables[name] = newValue
	_, err := s.rumCommandTemplate(context, session, "export %v='%v'", name, newValue)
	return err
}

func (s *execService) changeDirectory(context *Context, session *SystemTerminalSession, commandInfo *CommandResponse, directory string) (string, error) {
	if directory == "" {
		return "", nil
	}
	parent, name := path.Split(directory)
	if path.Ext(name) != "" {
		directory = parent
	}
	if len(directory) > 1 && strings.HasSuffix(directory, "/") {
		directory = string(directory[:len(directory)-1])
	}
	if session.currentDirectory == directory {
		return "", nil
	}

	result, err := s.rumCommandTemplate(context, session, "cd %v", directory)
	if err != nil {
		return "", err
	}

	if !CheckNoSuchFileOrDirectory(result) {
		session.currentDirectory = directory
	}
	return result, err
}

func (s *execService) rumCommandTemplate(context *Context, session *SystemTerminalSession, commandTemplate string, arguments ...interface{}) (string, error) {
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
		return stdout, err
	}
	return stdout, nil
}

func (s *execService) applyCommandOptions(context *Context, options *ExecutionOptions, session *SystemTerminalSession, info *CommandResponse) error {
	operatingSystem := session.OperatingSystem
	if options == nil {
		return nil
	}
	if len(options.SystemPaths) > 0 {
		operatingSystem.Path.Push(options.SystemPaths...)
	}
	err := s.setEnvVariables(context, session, options.Env)
	if err != nil {
		return err
	}
	if options.Directory != "" {
		directory := context.Expand(options.Directory)
		_, err := s.changeDirectory(context, session, info, directory)
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
		if escapedContains(stdout, candidate) {
			return candidate
		}
	}
	return ""
}

//TODO caching this
func (s *execService) credential(key string, credentialURI string) (result string, err error) {
	s.mutex.RLock()
	credConfig, has := s.credentials[credentialURI]
	s.mutex.RUnlock()
	result = key
	var getCredentail = func(key string, config *cred.Config) string {
		if strings.HasPrefix(key, "#") {
			return config.Username
		}
		return config.Password
	}
	if has {
		return getCredentail(key, credConfig), nil
	}
	credConfig = &cred.Config{}
	if credentialURI != "" && toolbox.FileExists(credentialURI) {
		credConfig, err = cred.NewConfig(credentialURI)
		if err != nil {
			return "", err
		}
		s.mutex.Lock()
		s.credentials[credentialURI] = credConfig
		s.mutex.Unlock()
		result = getCredentail(key, credConfig)
	}
	return result, nil
}

func (s *execService) credentialsToSecure(credentials map[string]string) (map[string]string, error) {
	var secure = make(map[string]string)
	if len(credentials) > 0 {
		for k, v := range credentials {
			secure[k] = v
			if toolbox.IsCompleteJSON(v) {
				continue
			}

			var credential, err = s.credential(k, v)
			if err != nil {
				return nil, err
			}
			secure[k] = credential

		}
	}
	return secure, nil
}

func (s *execService) validateStdout(stdout string, command string, execution *Execution) error {
	errorMatch := match(stdout, execution.Errors...)
	if errorMatch != "" {
		return fmt.Errorf("encounter error fragment: (%v), command:%v, stdout: %v", errorMatch, command, stdout)
	}
	if len(execution.Success) > 0 {
		sucessMatch := match(stdout, execution.Success...)
		if sucessMatch == "" {
			return fmt.Errorf("failed to match any fragment: '%v', command: %v; stdout: %v", strings.Join(execution.Success, ","), command, stdout)
		}
	}
	return nil
}

func (s *execService) executeCommand(context *Context, session *SystemTerminalSession, execution *Execution, options *ExecutionOptions, response *CommandResponse, request *ExtractableCommandRequest) error {
	command := context.Expand(execution.Command)

	terminators := getTerminators(options, session, execution)

	var cmd = command
	if len(execution.Credentials) > 0 {
		secure, err := s.credentialsToSecure(execution.Credentials)
		if err != nil {
			return fmt.Errorf("failed to run commend: %v, invalid credential: %v %v ", command, execution.Credentials, err)
		}
		var keys = toolbox.MapKeysToStringSlice(secure)
		sort.Strings(keys)
		for _, key := range keys {
			cmd = strings.Replace(cmd, key, secure[key], len(command))
			if strings.HasPrefix(key, "#") {
				command = strings.Replace(command, key, secure[key], len(command))
			}
		}
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

	response.Add(NewCommandLog(command, stdout, err))
	if err != nil {
		return err
	}

	if err = s.validateStdout(stdout, command, execution); err != nil {
		return err
	}

	err = execution.Extraction.Extract(context, response.Extracted, strings.Split(stdout, "\n")...)
	if err != nil {
		return err
	}

	if len(stdout) > 0 {
		for _, execution := range request.ExtractableCommand.Executions {
			if execution.MatchOutput != "" && strings.Contains(stdout, execution.MatchOutput) {
				return s.executeCommand(context, session, execution, options, response, request)
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
	terminators = append(terminators, execution.Errors...)
	return terminators
}

func (s *execService) runCommands(context *Context, request *CommandRequest) (*CommandResponse, error) {
	var mangedCommandRequest = request.AsExtractableCommandRequest()
	if request.SuperUser {
		superCommandRequest := SuperUserCommandRequest{
			Target:        request.Target,
			MangedCommand: mangedCommandRequest.ExtractableCommand,
		}
		var err error
		if mangedCommandRequest, err = superCommandRequest.AsCommandRequest(context); err != nil {
			return nil, err
		}
	}
	return s.runCommandsAndExtractData(context, mangedCommandRequest)
}

func (s *execService) runCommandsAndExtractData(context *Context, request *ExtractableCommandRequest) (*CommandResponse, error) {
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

	var options = request.ExtractableCommand.Options
	if options == nil {
		options = NewExecutionOptions()
	}
	response := NewCommandResponse(session.ID)
	err = s.applyCommandOptions(context, options, session, response)

	if err != nil {
		return nil, err
	}

	operatingSystem := session.OperatingSystem
	err = s.setEnvVariable(context, session, "PATH", operatingSystem.Path.EnvValue())
	if err != nil {
		return nil, err
	}

	response = NewCommandResponse(session.ID)
	for _, execution := range request.ExtractableCommand.Executions {
		var command = context.Expand(execution.Command)
		if execution.MatchOutput != "" {
			continue
		}
		if strings.HasPrefix(command, "cd ") {
			if !strings.Contains(command, "&&") {
				var directory = strings.TrimSpace(string(command[3:]))
				stdout, err := s.changeDirectory(context, session, response, directory)
				if err == nil {
					err = s.validateStdout(stdout, execution.Command, execution)
				}
				if err != nil {
					return nil, err
				}
				continue
			}
			session.currentDirectory = "" //reset path
		}
		if strings.HasPrefix(command, "export ") {
			if !strings.Contains(command, "&&") {
				envVariable := string(command[7:])
				keyValuePair := strings.Split(envVariable, "=")
				if len(keyValuePair) == 2 {
					key := strings.TrimSpace(keyValuePair[0])
					value := strings.TrimSpace(keyValuePair[1])
					value = strings.Trim(value, "'\"")
					err = s.setEnvVariable(context, session, key, value)
					continue
				}
			}
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

func isAmd64Architecture(candidate string) bool {
	return strings.Contains(candidate, "amd64") || strings.Contains(candidate, "x86_64")
}

func (s *execService) detectOperatingSystem(session *SystemTerminalSession) (*OperatingSystem, error) {
	operatingSystem := &OperatingSystem{
		Path: &SystemPath{
			SystemPath: make([]string, 0),
			Path:       make([]string, 0),
			index:      make(map[string]bool),
		},
	}

	varsionCheckCommand := "lsb_release -a"
	if session.MultiCommandSession.System() == "darwin" {
		varsionCheckCommand = "sw_vers"
	}
	output, err := session.Run(varsionCheckCommand, 0)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(output, "\r\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if isAmd64Architecture(line) {
			operatingSystem.Architecture = "amd64"
		}
		pair := strings.Split(line, ":")
		if len(pair) != 2 {
			continue
		}

		var key = strings.Replace(strings.ToLower(pair[0]), " ", "", len(pair[0]))
		var val = strings.Replace(strings.Trim(pair[1], " \t\r"), " ", "", len(line))
		switch key {
		case "productname", "distributorid":
			operatingSystem.Name = strings.ToLower(val)
		case "productversion", "release":
			operatingSystem.Version = strings.ToLower(val)
		}

	}
	operatingSystem.Hardware, err = session.Run("uname -m", 0)
	if err != nil {
		return nil, err
	}
	if isAmd64Architecture(operatingSystem.Hardware) {
		operatingSystem.Architecture = "amd64"
	}
	operatingSystem.System = session.System()
	output, err = session.Run("echo $PATH", 0)
	if err != nil {
		return nil, err
	}
	lines = strings.Split(output, "\r\n")
	for i := 0; i < len(lines); i++ {
		var line = lines[i]
		if !strings.Contains(line, ":") || !strings.Contains(line, "/") {
			continue
		}
		operatingSystem.Path.SystemPath = strings.Split(line, ":")
		break

	}
	return operatingSystem, nil
}

const (
	execServiceOpenExample = `{
  "Target": {
    "URL": "scp://127.0.0.1/",
    "Credential": "${env.HOME}/.secret/localhost.json"
  },
  
  "SystemPaths": ["/usr/local/bin"],
  "Env": {
    "GOPATH":"${env.HOME}/go"
  }
}`
	execServiceRunExample = `{
  "Target": {
    "URL": "scp://127.0.0.1/",
    "Credential": "${env.HOME}/.secret/localhost.json"
  },
  "Commands":["mkdir /tmp/app1"]
}`

	execServiceRunAndExtractExample = `{
  "Target": {
    "URL": "scp://127.0.0.1/",
    "Credential": "${env.HOME}/.secret/localhost.json"
  },
  "ExtractableCommand": {
    "Options": {
      "SystemPaths": [
        "/opt/sdk/go/bin"
      ]
    },
    "Executions": [
      {
        "Command": "go version",
        "Extraction": [
          {
            "RegExpr": "go(\\d\\.\\d)",
            "Key": "Version"
          }
        ]
      }
    ]
  }
}`

	execServiceManagedCloseExample = `{
  "Target": {
    "URL": "scp://127.0.0.1/",
    "Credential": "${env.HOME}/.secret/localhost.json"
  }
}`
)

func (s *execService) registerRoutes() {

	s.Register(&ServiceActionRoute{
		Action: "open",
		RequestInfo: &ActionInfo{
			Description: "open SSH session, usually no need for using this action directly since run,extract actions open session if needed",
			Examples: []*ExampleUseCase{
				{
					UseCase: "open session",
					Data:    execServiceOpenExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &OpenSessionRequest{}
		},
		ResponseProvider: func() interface{} {
			return &OpenSessionResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*OpenSessionRequest); ok {
				return s.open(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "run",
		RequestInfo: &ActionInfo{
			Description: "run terminal command",

			Examples: []*ExampleUseCase{
				{
					UseCase: "run command",
					Data:    execServiceRunExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &CommandRequest{}
		},
		ResponseProvider: func() interface{} {
			return &CommandResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*CommandRequest); ok {
				return s.runCommands(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "extract",
		RequestInfo: &ActionInfo{
			Description: "run terminal command and extract data from the stdout",

			Examples: []*ExampleUseCase{
				{
					UseCase: "run and extract command",
					Data:    execServiceRunAndExtractExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &ExtractableCommandRequest{}
		},
		ResponseProvider: func() interface{} {
			return &CommandResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*ExtractableCommandRequest); ok {
				return s.runCommandsAndExtractData(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "sudo",
		RequestInfo: &ActionInfo{
			Description: "run terminal command and extract data as rooot",
		},
		RequestProvider: func() interface{} {
			return &SuperUserCommandRequest{}
		},
		ResponseProvider: func() interface{} {
			return &CommandResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*SuperUserCommandRequest); ok {
				commandRequest, err := handlerRequest.AsCommandRequest(context)
				if err != nil {
					return nil, err
				}
				return s.runCommandsAndExtractData(context, commandRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "close",
		RequestInfo: &ActionInfo{
			Description: "close SSH terminal session, if created by run or extract it is scheduled to be closed at the end of endly run context.Close()",

			Examples: []*ExampleUseCase{
				{
					UseCase: "close ",
					Data:    execServiceManagedCloseExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &CloseSessionRequest{}
		},
		ResponseProvider: func() interface{} {
			return &CloseSessionResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*CloseSessionRequest); ok {
				return s.closeSession(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//NewExecService creates a new execution service
func NewExecService() Service {
	var result = &execService{
		mutex:           &sync.RWMutex{},
		credentials:     make(map[string]*cred.Config),
		AbstractService: NewAbstractService(ExecServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}

//SuperUserCommandRequest represents a super user command,
type SuperUserCommandRequest struct {
	Target        *url.Resource       //target destination where to run a command.
	MangedCommand *ExtractableCommand //managed command
}

//AsCommandRequest returns ExtractableCommandRequest
func (r *SuperUserCommandRequest) AsCommandRequest(context *Context) (*ExtractableCommandRequest, error) {
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
		Terminators: []string{"Password", commandNotFound},
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
	var extractions = make([]*DataExtraction, 0)

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
	credentials[sudoCredentialKey] = target.Credential
	execution := &Execution{
		Credentials: credentials,
		MatchOutput: "Password",
		Command:     sudoCredentialKey,
		Errors:      []string{"Password", commandNotFound},
		Extraction:  extractions,
	}
	execution.Errors = append(execution.Errors, errors...)
	result.ExtractableCommand.Executions = append(result.ExtractableCommand.Executions, execution)
	return result, nil
}
