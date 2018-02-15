package endly

import (
	"fmt"
	"github.com/lunixbochs/vtclean"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"path"
	"strings"
)

//DaemonServiceID represents system daemon service
const DaemonServiceID = "daemon"

const (
	serviceTypeError = iota
	serviceTypeInitDaemon
	serviceTypeLaunchCtl
	serviceTypeStdService
	serviceTypeSystemctl
)

type daemonService struct {
	*AbstractService
}

func (s *daemonService) getDarwinLaunchServiceInfo(context *Context, target *url.Resource, request *DaemonStatusRequest, info *DaemonInfo) error {

	if request.Exclusion != "" {
		request.Exclusion = " | grep -v " + request.Exclusion
	}
	commandResult, err := context.Execute(target, &ExtractableCommand{
		Executions: []*Execution{
			{
				Command: fmt.Sprintf("ls /Library/LaunchDaemons/ | grep %v %v", request.Service, request.Exclusion),
			},
		},
	})
	if err != nil {
		return err
	}
	file := strings.TrimSpace(commandResult.Stdout())
	if len(file) > 0 {
		info.Path = path.Join("/Library/LaunchDaemons/", file)
	}

	commandResult, err = context.Execute(target, &ExtractableCommand{
		Executions: []*Execution{
			{
				Command: fmt.Sprintf("launchctl list | grep %v %v", request.Service, request.Exclusion),
			},
		},
	})
	if err != nil {
		return err
	}
	stdout := commandResult.Stdout()
	for _, line := range strings.Split(stdout, "\n") {
		columns, ok := ExtractColumns(line)
		if !ok || len(columns) == 0 {
			continue
		}
		var pid = toolbox.AsInt(columns[0])
		if info.Pid > 0 && pid == 0 {
			continue
		}
		info.Pid = pid
		info.Domain = columns[len(columns)-1]
		info.Launched = true
	}
	return nil
}

func (s *daemonService) determineServiceType(context *Context, service, exclusion string, target *url.Resource) (int, error) {
	session, err := context.TerminalSession(target)
	if err != nil {
		return 0, err
	}
	if session.DaemonType > 0 {
		return session.DaemonType, nil
	}

	var systemTypeCommands = []struct {
		systemType int
		command    string
	}{
		{serviceTypeLaunchCtl, "launchctl version"},
		{serviceTypeStdService, "service --version"},
		{serviceTypeSystemctl, "systemctl --version"},
	}
	var commandResult *CommandResponse
	for _, candidate := range systemTypeCommands {
		commandResult, err = context.Execute(target, &ExtractableCommand{
			Executions: []*Execution{
				{
					Command: candidate.command,
				},
			},
		})
		if err != nil {
			break
		}
		var stdout = commandResult.Stdout()
		if !CheckNoSuchFileOrDirectory(stdout) && !CheckCommandNotFound(stdout) {
			session.DaemonType = candidate.systemType
			return session.DaemonType, nil
		}
	}
	return serviceTypeError, nil
}

func extractServiceInfo(stdout string, state map[string]string, info *DaemonInfo) {
	if pid, ok := state["pid"]; ok {
		info.Pid = toolbox.AsInt(pid)
	}
	if value, ok := state["state"]; ok {
		state := vtclean.Clean(value, false)
		if strings.Contains(state, "inactive") {
			state = "not running"
		} else if strings.Contains(state, "active") {
			state = "running"
		}
		info.State = state
	}
	if daemonPath, ok := state["path"]; ok {
		info.Path = daemonPath
	}

	//deal with service deamon		info.Pid = docker start/running, process 48628
	if info.State == "" {
		candidate := vtclean.Clean(stdout, false)
		if strings.Contains(candidate, "start/running") {
			info.State = "running"
			if columns, ok := ExtractColumns(info.State); ok {
				if len(columns) > 0 {
					info.Pid = toolbox.AsInt(columns[len(columns)-1])
				}
			}
		} else if strings.Contains(candidate, "stop/waiting") {
			info.State = "not running"
		}
	}
}

func (s *daemonService) executeCommand(context *Context, serviceType int, target *url.Resource, command *ExtractableCommand) (*CommandResponse, error) {
	if serviceType == serviceTypeLaunchCtl {
		return context.Execute(target, command)
	}
	return context.ExecuteAsSuperUser(target, command)
}

func (s *daemonService) isLaunchCtlDomainMissing(info *DaemonInfo) bool {
	return (info.Path == "" || info.Domain == "") && info.Type == serviceTypeLaunchCtl
}

func (s *daemonService) determineCheckCommand(context *Context, target *url.Resource, serviceType int, info *DaemonInfo) (command string, err error) {
	switch serviceType {
	case serviceTypeError:
		return "", fmt.Errorf("unknown daemon service type")

	case serviceTypeLaunchCtl:

		if info.Pid > 0 {
			commandResult, err := context.ExecuteAsSuperUser(target, &ExtractableCommand{
				Executions: []*Execution{
					{
						Command: fmt.Sprintf("launchctl procinfo %v", info.Pid),
						Extraction: DataExtractions{
							{
								Key:     "path",
								RegExpr: "program path[\\s|\\t]+=[\\s|\\t]+([^\\s]+)",
							},
							{
								Key:     "state",
								RegExpr: "state = (running)",
							},
						},
						Errors: []string{"Unrecognized"},
					},
				},
			})
			if err != nil {
				return "", err
			}
			extractServiceInfo(commandResult.Stdout(), commandResult.Extracted, info)
		}
		return "", nil

	case serviceTypeSystemctl:
		command = fmt.Sprintf("systemctl status %v ", info.Service)
	case serviceTypeStdService:
		command = fmt.Sprintf("service %v status", info.Service)
	case serviceTypeInitDaemon:
		command = fmt.Sprintf("%v status", info.Service)
	}
	return command, nil
}

func (s *daemonService) checkService(context *Context, request *DaemonStatusRequest) (*DaemonInfo, error) {
	if request.Service == "" {
		return nil, fmt.Errorf("service was empty")
	}
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}

	serviceType, err := s.determineServiceType(context, request.Service, request.Exclusion, target)
	if err != nil {
		return nil, err
	}
	var info = &DaemonInfo{
		Service: request.Service,
		Type:    serviceType,
	}

	if serviceType == serviceTypeLaunchCtl {
		err = s.getDarwinLaunchServiceInfo(context, target, request, info)
		if err != nil {
			return nil, err
		}
	}
	command := ""
	if s.isLaunchCtlDomainMissing(info) {
		return info, nil
	}

	command, err = s.determineCheckCommand(context, target, serviceType, info)
	if err != nil {
		return nil, err
	}
	if command == "" {
		return info, err
	}

	commandResult, err := s.executeCommand(context, serviceType, target, &ExtractableCommand{
		Options: &ExecutionOptions{
			Terminators: []string{"(END)"},
		},
		Executions: []*Execution{

			{
				Command: command,
				Extraction: DataExtractions{
					{
						Key:     "pid",
						RegExpr: "[^└]+└─(\\d+).+",
					},
					{
						Key:     "pid",
						RegExpr: " Main PID: (\\d+).+",
					},
					{
						Key:     "state",
						RegExpr: "[\\s|\\t]+Active:\\s+(\\S+)",
					},
					{
						Key:     "path",
						RegExpr: "[^└]+└─\\d+[\\s\\t].(.+)",
					},
				},
			},
			{
				MatchOutput: "(END)", //quite multiline mode
				Command:     "Q",
			},
		},
	})
	if err != nil {
		return nil, err
	}

	extractServiceInfo(commandResult.Stdout(), commandResult.Extracted, info)
	return info, nil

}

func (s *daemonService) stopService(context *Context, request *DaemonStopRequest) (*DaemonStopResponse, error) {
	serviceInfo, err := s.checkService(context, &DaemonStatusRequest{
		Target:    request.Target,
		Service:   request.Service,
		Exclusion: request.Exclusion,
	})
	if err != nil {
		return nil, err
	}
	if !serviceInfo.IsActive() {
		return &DaemonStopResponse{serviceInfo}, nil
	}
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	command := ""
	switch serviceInfo.Type {
	case serviceTypeError:
		return nil, fmt.Errorf("Unknown daemon service type")
	case serviceTypeLaunchCtl:
		command = fmt.Sprintf("launchctl stop  %v", serviceInfo.Domain)
	case serviceTypeSystemctl:
		command = fmt.Sprintf("systemctl stop %v ", serviceInfo.Service)
	case serviceTypeStdService:
		command = fmt.Sprintf("service %v stop", serviceInfo.Service)
	case serviceTypeInitDaemon:
		command = fmt.Sprintf("%v stop", serviceInfo.Service)
	}
	commandResult, err := s.executeCommand(context, serviceInfo.Type, target, &ExtractableCommand{
		Executions: []*Execution{
			{
				Command: command,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if CheckCommandNotFound(commandResult.Stdout()) {
		return nil, fmt.Errorf("%v", commandResult.Stdout())
	}
	info, err := s.checkService(context, &DaemonStatusRequest{
		Target:    request.Target,
		Service:   request.Service,
		Exclusion: request.Exclusion,
	})
	return &DaemonStopResponse{info}, err
}

func (s *daemonService) startService(context *Context, request *DaemonStartRequest) (*DaemonStartResponse, error) {
	serviceInfo, err := s.checkService(context, &DaemonStatusRequest{
		Target:    request.Target,
		Service:   request.Service,
		Exclusion: request.Exclusion,
	})
	if err != nil {
		return nil, err
	}
	if serviceInfo.IsActive() {
		return &DaemonStartResponse{DaemonInfo: serviceInfo}, nil
	}
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	command := ""
	switch serviceInfo.Type {
	case serviceTypeError:
		return nil, fmt.Errorf("Unknown daemon service type")
	case serviceTypeLaunchCtl:
		if !serviceInfo.Launched {
			command = fmt.Sprintf("launchctl load -F %v", serviceInfo.Path)
			_, err = s.executeCommand(context, serviceInfo.Type, target, &ExtractableCommand{
				Executions: []*Execution{
					{
						Command: command,
					},
				},
			})
			if err != nil {
				return nil, err
			}
		}
		serviceInfo, _ = s.checkService(context, &DaemonStatusRequest{
			Target:    request.Target,
			Service:   request.Service,
			Exclusion: request.Exclusion,
		})
		command = fmt.Sprintf("launchctl start %v", serviceInfo.Domain)
	case serviceTypeSystemctl:
		command = fmt.Sprintf("systemctl start %v ", serviceInfo.Service)
	case serviceTypeStdService:
		command = fmt.Sprintf("service %v start", serviceInfo.Service)
	case serviceTypeInitDaemon:
		command = fmt.Sprintf("%v start", serviceInfo.Service)
	}

	commandResult, err := s.executeCommand(context, serviceInfo.Type, target, &ExtractableCommand{
		Executions: []*Execution{
			{
				Command: command,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if CheckCommandNotFound(commandResult.Stdout()) {
		return nil, fmt.Errorf("%v", commandResult.Stdout())
	}

	serviceInfo, err = s.checkService(context, &DaemonStatusRequest{
		Target:    request.Target,
		Service:   request.Service,
		Exclusion: request.Exclusion,
	})
	if serviceInfo != nil && !serviceInfo.IsActive() {
		return nil, fmt.Errorf("%v service is inactive", request.Service)
	}
	return &DaemonStartResponse{DaemonInfo: serviceInfo}, err
}

func (s *daemonService) registerRoutes() {
	s.Register(&ServiceActionRoute{
		Action: "start",
		RequestInfo: &ActionInfo{
			Description: "start provided service on target host",
		},
		RequestProvider: func() interface{} {
			return &DaemonStartRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DaemonStartResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*DaemonStartRequest); ok {
				return s.startService(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
	s.Register(&ServiceActionRoute{
		Action: "status",
		RequestInfo: &ActionInfo{
			Description: "check service status on target host",
		},
		RequestProvider: func() interface{} {
			return &DaemonStatusRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DaemonInfo{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*DaemonStatusRequest); ok {
				return s.checkService(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
	s.Register(&ServiceActionRoute{
		Action: "stop",
		RequestInfo: &ActionInfo{
			Description: "stop service on target host",
		},
		RequestProvider: func() interface{} {
			return &DaemonStopRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DaemonInfo{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*DaemonStopRequest); ok {
				return s.stopService(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

}

//NewDaemonService creates a new system service.
func NewDaemonService() Service {
	var result = &daemonService{
		AbstractService: NewAbstractService(DaemonServiceID),
	}
	result.registerRoutes()
	result.AbstractService.Service = result
	return result
}
