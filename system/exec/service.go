package exec

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/ssh"
	"github.com/viant/toolbox/url"
	"path"
	"sort"
	"strings"
)

//ServiceID represent system executor service id
const ServiceID = "exec"

//SudoCredentialKey represent obsucated password sudo credential key (target.Credential)
const SudoCredentialKey = "**sudo**"

type execService struct {
	*endly.AbstractService
	credentials map[string]*cred.Config
}

func (s *execService) open(context *endly.Context, request *OpenSessionRequest) (*OpenSessionResponse, error) {
	var clientSession, err = s.openSession(context, request)
	if err != nil {
		return nil, err
	}
	return &OpenSessionResponse{
		SessionID: clientSession.ID,
	}, nil
}

func (s *execService) openSSHService(context *endly.Context, request *OpenSessionRequest) (ssh.Service, error) {
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
	hostname, port := s.GetHostAndSSHPort(target)
	return ssh.NewService(hostname, port, authConfig)
}

func (s *execService) isSupportedScheme(target *url.Resource) bool {
	return target.ParsedURL.Scheme == "ssh" || target.ParsedURL.Scheme == "scp" || target.ParsedURL.Scheme == "file"
}

func (s *execService) initSession(context *endly.Context, target *url.Resource, session *endly.SystemTerminalSession, env map[string]string) error {
	_, _ = s.changeDirectory(context, session, nil, target.ParsedURL.Path)
	for k, v := range env {
		if err := s.setEnvVariable(context, session, k, v); err != nil {
			return err
		}
	}
	return nil
}

func (s *execService) captureCommandIfNeeded(context *endly.Context, replayCommands *ssh.ReplayCommands, sshService ssh.Service) (err error) {
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

func (s *execService) openSession(context *endly.Context, request *OpenSessionRequest) (*endly.SystemTerminalSession, error) {
	s.Lock()
	defer s.Unlock()
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
	session, err := endly.NewSystemTerminalSession(sessionName, sshService)
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

func (s *execService) setEnvVariables(context *endly.Context, session *endly.SystemTerminalSession, env map[string]string) error {
	for k, v := range env {
		err := s.setEnvVariable(context, session, k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *execService) setEnvVariable(context *endly.Context, session *endly.SystemTerminalSession, name, newValue string) error {
	newValue = context.Expand(newValue)

	if actual, has := session.EnvVariables[name]; has {
		if newValue == actual {
			return nil
		}
	}
	session.EnvVariables[name] = newValue
	_, err := s.rumCommandTemplate(context, session, "export %v='%v'", name, newValue)
	return err
}

func (s *execService) changeDirectory(context *endly.Context, session *endly.SystemTerminalSession, commandInfo *RunResponse, directory string) (string, error) {
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
	if session.CurrentDirectory == directory {
		return "", nil
	}

	result, err := s.rumCommandTemplate(context, session, "cd %v", directory)
	if err != nil {
		return "", err
	}

	if !util.CheckNoSuchFileOrDirectory(result) {
		session.CurrentDirectory = directory
	}
	return result, err
}

func (s *execService) rumCommandTemplate(context *endly.Context, session *endly.SystemTerminalSession, commandTemplate string, arguments ...interface{}) (string, error) {
	command := fmt.Sprintf(commandTemplate, arguments...)
	startEvent := s.Begin(context, NewSdtinEvent(session.ID, command))
	stdout, err := session.Run(command, 1000)
	s.End(context)(startEvent, NewStdoutEvent(session.ID, stdout, err))
	return stdout, err
}

func (s *execService) applyCommandOptions(context *endly.Context, options *ExecutionOptions, session *endly.SystemTerminalSession, info *RunResponse) error {
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
		if util.EscapedContains(stdout, candidate) {
			return candidate
		}
	}
	return ""
}

//TODO caching this
func (s *execService) credential(key string, credentialURI string) (result string, err error) {
	s.RLock()
	credConfig, has := s.credentials[credentialURI]
	s.RUnlock()
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
	if credentialURI != "" {
		credConfig, err = cred.NewConfig(credentialURI)
		if err != nil {
			return "", err
		}
		s.Lock()
		s.credentials[credentialURI] = credConfig
		s.Unlock()
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

func (s *execService) executeCommand(context *endly.Context, session *endly.SystemTerminalSession, execution *Execution, options *ExecutionOptions, response *RunResponse, request *ExtractRequest) error {
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
	startEvent := s.Begin(context, NewSdtinEvent(session.ID, command))
	stdout, err := session.Run(cmd, options.TimeoutMs, terminators...)
	s.End(context)(startEvent, NewStdoutEvent(session.ID, stdout, err))
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
func getTerminators(options *ExecutionOptions, session *endly.SystemTerminalSession, execution *Execution) []string {
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

func (s *execService) runCommands(context *endly.Context, request *RunRequest) (*RunResponse, error) {
	var mangedCommandRequest = request.AsExtractRequest()
	if request.SuperUser {
		superCommandRequest := SuperRunRequest{
			Target:        request.Target,
			MangedCommand: mangedCommandRequest.ExtractableCommand,
		}
		var err error
		if mangedCommandRequest, err = superCommandRequest.AsExtractRequest(context); err != nil {
			return nil, err
		}
	}
	return s.runCommandsAndExtractData(context, mangedCommandRequest)
}

func (s *execService) runCommandsAndExtractData(context *endly.Context, request *ExtractRequest) (*RunResponse, error) {
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
	response := NewRunResponse(session.ID)
	err = s.applyCommandOptions(context, options, session, response)

	if err != nil {
		return nil, err
	}

	operatingSystem := session.OperatingSystem
	err = s.setEnvVariable(context, session, "PATH", operatingSystem.Path.EnvValue())
	if err != nil {
		return nil, err
	}

	response = NewRunResponse(session.ID)
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
			session.CurrentDirectory = "" //reset path
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
			session.EnvVariables = make(map[string]string) //reset env variables
		}
		err = s.executeCommand(context, session, execution, options, response, request)
		if err != nil {
			return nil, err
		}

	}
	return response, nil
}

func (s *execService) closeSession(context *endly.Context, request *CloseSessionRequest) (*CloseSessionResponse, error) {
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

func (s *execService) detectOperatingSystem(session *endly.SystemTerminalSession) (*endly.OperatingSystem, error) {
	operatingSystem := &endly.OperatingSystem{
		Path: endly.NewSystemPath(),
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
		operatingSystem.Path = endly.NewSystemPath(strings.Split(line, ":")...)
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

	s.Register(&endly.ServiceActionRoute{
		Action: "open",
		RequestInfo: &endly.ActionInfo{
			Description: "open SSH session, usually no need for using this action directly since run,extract actions open session if needed",
			Examples: []*endly.ExampleUseCase{
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
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*OpenSessionRequest); ok {
				return s.open(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "run",
		RequestInfo: &endly.ActionInfo{
			Description: "run terminal command",

			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "run command",
					Data:    execServiceRunExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &RunRequest{}
		},
		ResponseProvider: func() interface{} {
			return &RunResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*RunRequest); ok {
				return s.runCommands(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "extract",
		RequestInfo: &endly.ActionInfo{
			Description: "run terminal command and extract data from the stdout",

			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "run and extract command",
					Data:    execServiceRunAndExtractExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &ExtractRequest{}
		},
		ResponseProvider: func() interface{} {
			return &RunResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*ExtractRequest); ok {
				return s.runCommandsAndExtractData(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "sudo",
		RequestInfo: &endly.ActionInfo{
			Description: "run terminal command and extract data as rooot",
		},
		RequestProvider: func() interface{} {
			return &SuperRunRequest{}
		},
		ResponseProvider: func() interface{} {
			return &RunResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*SuperRunRequest); ok {
				commandRequest, err := req.AsExtractRequest(context)
				if err != nil {
					return nil, err
				}
				return s.runCommandsAndExtractData(context, commandRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "close",
		RequestInfo: &endly.ActionInfo{
			Description: "close SSH terminal session, if created by run or extract it is scheduled to be closed at the end of endly run context.Close()",

			Examples: []*endly.ExampleUseCase{
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
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*CloseSessionRequest); ok {
				return s.closeSession(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//New creates a new execution service
func New() endly.Service {
	var result = &execService{
		credentials:     make(map[string]*cred.Config),
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
