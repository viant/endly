package exec

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/criteria"
	"github.com/viant/endly/model"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/secret"
	"github.com/viant/toolbox/ssh"
	"github.com/viant/toolbox/url"
	"path"
	"strings"
)

//ServiceID represent system executor service id
const ServiceID = "exec"

//SudoCredentialKey represent obsucated password sudo credentials key (target.Credentials)
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
	authConfig, err := context.Secrets.GetOrCreate(target.Credentials)
	if err != nil {
		return nil, err
	}
	hostname, port := s.GetHostAndSSHPort(target)
	return ssh.NewService(hostname, port, authConfig)
}

func (s *execService) isSupportedScheme(target *url.Resource) bool {
	return target.ParsedURL.Scheme == "ssh" || target.ParsedURL.Scheme == "scp" || target.ParsedURL.Scheme == "file"
}

func (s *execService) initSession(context *endly.Context, target *url.Resource, session *model.Session, env map[string]string) error {
	_, _ = s.changeDirectory(context, session, nil, target.ParsedURL.Path)
	for k, v := range env {
		if err := s.setEnvVariable(context, session, k, v); err != nil {
			return err
		}
	}
	return nil
}

func (s *execService) openSession(context *endly.Context, request *OpenSessionRequest) (*model.Session, error) {
	s.Lock()
	defer s.Unlock()
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	if !s.isSupportedScheme(target) {
		return nil, fmt.Errorf("failed to open sessionID: invalid schema: %v in url: %v", target.ParsedURL.Scheme, target.URL)
	}
	sessions := TerminalSessions(context)

	var replayCommands *ssh.ReplayCommands
	if request.Basedir != "" {
		replayCommands, err = ssh.NewReplayCommands(request.Basedir)
		if err != nil {
			return nil, err
		}
	}
	var sessionID = target.Host()
	if sessions.Has(sessionID) {
		SShSession := sessions[sessionID]
		err = s.initSession(context, target, SShSession, request.Env)
		if err != nil {
			return nil, err
		}
		return sessions[sessionID], err
	}
	sshService, err := s.openSSHService(context, request)
	if err == nil {
		err = s.captureCommandIfNeeded(context, replayCommands, sshService)
	}
	if err != nil {
		return nil, err
	}
	SSHSession, err := model.NewSession(sessionID, sshService)
	if err != nil {
		return nil, err
	}
	if !request.Transient {
		context.Deffer(func() {
			_ = sshService.Close()
		})
	}
	SSHSession.MultiCommandSession, err = SSHSession.Service.OpenMultiCommandSession(request.Config)
	if err != nil {
		return nil, err
	}
	if !request.Transient {
		context.Deffer(func() {
			s.closeSession(context, &CloseSessionRequest{
				SessionID: sessionID,
			})
		})
	}
	err = s.initSession(context, target, SSHSession, request.Env)
	if err != nil {
		return nil, err
	}
	sessions[sessionID] = SSHSession
	SSHSession.Os, err = s.detectOperatingSystem(SSHSession)
	if err != nil {
		return nil, err
	}
	return SSHSession, nil
}

func (s *execService) setEnvVariables(context *endly.Context, session *model.Session, env map[string]string) error {
	for k, v := range env {
		err := s.setEnvVariable(context, session, k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *execService) setEnvVariable(context *endly.Context, session *model.Session, name, newValue string) error {
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

func (s *execService) changeDirectory(context *endly.Context, session *model.Session, commandInfo *RunResponse, directory string) (string, error) {
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

func (s *execService) run(context *endly.Context, session *model.Session, command string, listener ssh.Listener, timeoutMs int, terminators ...string) (stdout string, err error) {
	if stdout, err = session.Run(command, listener, timeoutMs, terminators...); err == nil {
		return stdout, err
	}
	if err == ssh.ErrTerminated {
		err := session.Reconnect()
		if err != nil {
			return "", err
		}
		currentDirectory := session.CurrentDirectory
		env := session.EnvVariables
		session.EnvVariables = make(map[string]string)
		session.CurrentDirectory = ""
		for k, v := range env {
			s.setEnvVariable(context, session, k, v)
		}
		runResponse := &RunResponse{}
		s.changeDirectory(context, session, runResponse, currentDirectory)
		return session.Run(command, listener, timeoutMs, terminators...)
	}
	return stdout, err
}

func (s *execService) rumCommandTemplate(context *endly.Context, session *model.Session, commandTemplate string, arguments ...interface{}) (string, error) {
	command := fmt.Sprintf(commandTemplate, arguments...)
	startEvent := s.Begin(context, NewSdtinEvent(session.ID, command))
	stdout, err := session.Run(command, nil, 1000)
	s.End(context)(startEvent, NewStdoutEvent(session.ID, stdout, err))
	return stdout, err
}

func (s *execService) applyCommandOptions(context *endly.Context, options *Options, session *model.Session, info *RunResponse) error {
	if len(options.SystemPaths) > 0 {
		session.Path.Push(options.SystemPaths...)
		if err := s.setEnvVariable(context, session, "PATH", session.Path.EnvValue()); err != nil {
			return err
		}
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

func (s *execService) commandAsSuperUser(session *model.Session, command string) string {
	if session.Username == "root" {
		return command
	}
	if len(command) > 1 && !strings.Contains(command, "sudo") {
		return "sudo " + command
	}
	return command
}

func (s *execService) validateStdout(stdout string, command string, execution *ExtractCommand) error {
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

func (s *execService) authSuperUserIfNeeded(stdout string, context *endly.Context, session *model.Session, extractCommand *ExtractCommand, response *RunResponse, request *ExtractRequest) (err error) {
	if !request.SuperUser || session.SuperUSerAuth {
		return nil
	}
	if util.EscapedContains(stdout, "Password") {
		session.SuperUSerAuth = true
		if len(request.Secrets) == 0 {
			request.Secrets = secret.NewSecrets(nil)
			request.Secrets[SudoCredentialKey] = secret.Secret(request.Target.Credentials)
		}
		extractCommand := NewExtractCommand(SudoCredentialKey, "", nil, []string{"Password", util.CommandNotFound})
		err = s.executeCommand(context, session, extractCommand, response, request)
	}
	return err
}

func (s *execService) buildMatchingState(response *RunResponse, context *endly.Context) data.Map {
	var state = context.State()
	var result = state.Clone()
	var logs = data.NewCollection()
	for _, log := range response.Cmd {
		var cmd = data.NewMap()
		cmd.Put("stdin", log.Stdin)
		cmd.Put("stdout", log.Stdout)
		logs.Push(cmd)
	}
	result.Put("cmd", logs)
	result.Put("output", response.Output)

	var stdout = ""
	if len(response.Cmd) > 0 {
		stdout = response.Cmd[len(response.Cmd)-1].Stdout
	}
	result.Put("stdout", stdout)
	return result
}

func (s *execService) executeCommand(context *endly.Context, session *model.Session, extractCommand *ExtractCommand, response *RunResponse, request *ExtractRequest) (err error) {
	var state = context.State()
	state.SetValue("os.user", session.Username)
	command := context.Expand(extractCommand.Command)
	options := request.Options
	terminators := getTerminators(options, session, extractCommand)
	if extractCommand.When != "" {
		var state = s.buildMatchingState(response, context)
		if ok, err := criteria.Evaluate(context, state, extractCommand.When, "Cmd.When", true); !ok {
			return err
		}
	}
	if request.SuperUser {
		if !session.SuperUSerAuth {
			terminators = append(terminators, "Password")
		}
		command = s.commandAsSuperUser(session, command)
	}
	var cmd = command

	if cmd, err = context.Secrets.Expand(cmd, request.Secrets); err != nil {
		return err
	}
	var listener ssh.Listener
	s.Begin(context, NewSdtinEvent(session.ID, command))
	listener = func(stdout string, hasMore bool) {
		context.Publish(NewStdoutEvent(session.ID, stdout, err))
	}
	stdout, err := s.run(context, session, cmd, listener, options.TimeoutMs, terminators...)
	if len(response.Output) > 0 {
		response.Output += "\n"
	}
	response.Output += stdout
	response.Add(NewCommandLog(command, stdout, err))
	if err != nil {
		return err
	}
	if err = s.validateStdout(stdout, command, extractCommand); err != nil {
		return err
	}
	err = s.authSuperUserIfNeeded(stdout, context, session, extractCommand, response, request)
	if err != nil {
		return err
	}
	stdout = response.Cmd[len(response.Cmd)-1].Stdout
	return extractCommand.Extraction.Extract(context, response.Data, strings.Split(stdout, "\n")...)
}

func getTerminators(options *Options, session *model.Session, execution *ExtractCommand) []string {
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
	return s.runExtractCommands(context, request.AsExtractRequest())
}

func (s *execService) runExtractCommands(context *endly.Context, request *ExtractRequest) (*RunResponse, error) {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	session, err := s.openSession(context, &OpenSessionRequest{Target: target})
	if err != nil {
		return nil, err
	}

	response := NewRunResponse(session.ID)
	if err = s.applyCommandOptions(context, request.Options, session, response); err != nil {
		return nil, err
	}

	response = NewRunResponse(session.ID)
	for _, extractCommand := range request.Commands {
		var command = context.Expand(extractCommand.Command)

		if strings.HasPrefix(command, "cd ") {
			if !strings.Contains(command, "&&") {
				var directory = strings.TrimSpace(string(command[3:]))
				stdout, err := s.changeDirectory(context, session, response, directory)
				if err == nil {
					err = s.validateStdout(stdout, command, extractCommand)
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
		err = s.executeCommand(context, session, extractCommand, response, request)
		if err != nil {
			return nil, err
		}

	}
	return response, nil
}

func (s *execService) closeSession(context *endly.Context, request *CloseSessionRequest) (*CloseSessionResponse, error) {
	clientSessions := TerminalSessions(context)
	if session, has := clientSessions[request.SessionID]; has {
		session.MultiCommandSession.Close()
		session.Close()
		delete(clientSessions, request.SessionID)
	}
	return &CloseSessionResponse{
		SessionID: request.SessionID,
	}, nil
}

func isAmd64Architecture(candidate string) bool {
	return strings.Contains(candidate, "amd64") || strings.Contains(candidate, "x86_64")
}

func (s *execService) extractOsPath(session *model.Session, os *model.OperatingSystem) error {
	output, err := session.Run("echo $PATH", nil, 0)
	if err != nil {
		return err
	}
	lines := strings.Split(output, "\r\n")
	for i := 0; i < len(lines); i++ {
		var line = lines[i]
		if !strings.Contains(line, ":") || !strings.Contains(line, "/") {
			continue
		}
		session.Path = model.NewPath(strings.Split(line, ":")...)
		break

	}
	return nil
}

func (s *execService) extractOsUser(session *model.Session, os *model.OperatingSystem) error {
	output, err := session.Run("echo $USER", nil, 0)
	if err != nil {
		return err
	}
	output = util.EscapeStdout(output)
	strings.Replace(output, "\n", "", len(output))
	session.Username = output
	return nil
}

func (s *execService) detectOperatingSystem(session *model.Session) (*model.OperatingSystem, error) {
	operatingSystem := &model.OperatingSystem{}
	session.Path = model.NewPath()

	varsionCheckCommand := "lsb_release -a"
	if session.MultiCommandSession.System() == "darwin" {
		varsionCheckCommand = "sw_vers"
	}
	output, err := session.Run(varsionCheckCommand, nil, 0)
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
	operatingSystem.Hardware, err = session.Run("uname -m", nil, 0)
	if err != nil {
		return nil, err
	}
	if isAmd64Architecture(operatingSystem.Hardware) {
		operatingSystem.Architecture = "amd64"
	}
	operatingSystem.System = session.System()
	if err = s.extractOsPath(session, operatingSystem); err == nil {
		err = s.extractOsUser(session, operatingSystem)
	}
	return operatingSystem, err
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

const (
	execServiceOpenExample = `{
  "Target": {
    "URL": "scp://127.0.0.1/",
    "Credentials": "${env.HOME}/.secret/localhost.json"
  },
  
  "SystemPaths": ["/usr/local/bin"],
  "Env": {
    "GOPATH":"${env.HOME}/go"
  }
}`
	execServiceRunExample = `{
  "Target": {
    "URL": "scp://127.0.0.1/",
    "Credentials": "${env.HOME}/.secret/localhost.json"
  },
  "Cmd":["mkdir /tmp/app1"]
}`

	execServiceRunAndExtractExample = `{
	"Target": {
	"URL": "scp://127.0.0.1/",
	"Credentials": "${env.HOME}/.secret/localhost.json"
	},
	"SystemPaths": [
	"/opt/sdk/go/bin"
	],
	"Cmd": [
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
}`

	execServiceManagedCloseExample = `{
  "Target": {
    "URL": "scp://127.0.0.1/",
    "Credentials": "${env.HOME}/.secret/localhost.json"
  }
}`
)

func (s *execService) registerRoutes() {

	s.Register(&endly.Route{
		Action: "open",
		RequestInfo: &endly.ActionInfo{
			Description: "open SSH session, usually no need for using this action directly since run,extract actions open session if needed",
			Examples: []*endly.UseCase{
				{
					Description: "open session",
					Data:        execServiceOpenExample,
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

	s.Register(&endly.Route{
		Action: "run",
		RequestInfo: &endly.ActionInfo{
			Description: "run terminal command",

			Examples: []*endly.UseCase{
				{
					Description: "run command",
					Data:        execServiceRunExample,
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

	s.Register(&endly.Route{
		Action: "extract",
		RequestInfo: &endly.ActionInfo{
			Description: "run terminal command and extract data from the stdout",

			Examples: []*endly.UseCase{
				{
					Description: "run and extract command",
					Data:        execServiceRunAndExtractExample,
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
				return s.runExtractCommands(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "close",
		RequestInfo: &endly.ActionInfo{
			Description: "close SSH terminal session, if created by run or extract it is scheduled to be closed at the end of endly run context.Close()",

			Examples: []*endly.UseCase{
				{
					Description: "close ",
					Data:        execServiceManagedCloseExample,
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
