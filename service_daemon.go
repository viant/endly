package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"path"
	"strings"
	"github.com/lunixbochs/vtclean"
)

//DaemonServiceID represents system daemon service
const DaemonServiceID = "daemon"

const (
	serviceTypeError      = iota
	serviceTypeInitDaemon
	serviceTypeLaunchCtl
	serviceTypeStdService
	serviceTypeSystemctl
)

type daemonService struct {
	*AbstractService
}

func (s *daemonService) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok"}
	defer s.End(context)(startEvent, Pairs("response", response))
	var err error
	switch actualRequest := request.(type) {
	case *DaemonStartRequest:
		response.Response, err = s.startService(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to start service: %v, %v", actualRequest.Service, err)
		}
	case *DaemonStopRequest:
		response.Response, err = s.stopService(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to stop service: %v, %v", actualRequest.Service, err)
		}
	case *DaemonStatusRequest:
		response.Response, err = s.checkService(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to check status service: %v, %v", actualRequest.Service, err)
		}
	}
	if response.Error != "" {
		response.Status = "err"
	}
	return response
}

func (s *daemonService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "status":
		return &DaemonStatusRequest{}, nil
	case "start":
		return &DaemonStartRequest{}, nil
	case "stop":
		return &DaemonStopRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

func (s *daemonService) getDarwinLaunchServiceInfo(context *Context, target *url.Resource, request *DaemonStatusRequest, info *DaemonInfo) (error) {

	if request.Exclusion != "" {
		request.Exclusion = " | grep -v " + request.Exclusion
	}
	commandResult, err := context.Execute(target, &ManagedCommand{
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

	commandResult, err = context.Execute(target, &ManagedCommand{
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
		if ! ok || len(columns) == 0 {
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
		commandResult, err = context.Execute(target, &ManagedCommand{
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
	fmt.Printf("NOT FOUND\n")

	return serviceTypeError, nil
}

func extractServiceInfo(state map[string]string, info *DaemonInfo) {
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
}

func (s *daemonService) executeCommand(context *Context, serviceType int, target *url.Resource, command *ManagedCommand) (*CommandResponse, error) {
	if serviceType == serviceTypeLaunchCtl {
		return context.Execute(target, command)
	}
	return context.ExecuteAsSuperUser(target, command)
}

func (s *daemonService) checkService(context *Context, request *DaemonStatusRequest) (*DaemonInfo, error) {
	if request.Service == "" {
		return nil, fmt.Errorf("Service was empty")
	}
	serviceType, err := s.determineServiceType(context, request.Service, request.Exclusion, request.Target)
	if err != nil {
		return nil, err
	}
	target, err := context.ExpandResource(request.Target)
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
	if (info.Path == "" || info.Domain == "" ) && serviceType == serviceTypeLaunchCtl {
		return info, nil
	}

	switch serviceType {
	case serviceTypeError:
		return nil, fmt.Errorf("Unknown daemon service type")

	case serviceTypeLaunchCtl:

		if info.Pid > 0 {
			commandResult, err := context.ExecuteAsSuperUser(target, &ManagedCommand{
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
						Error: []string{"Unrecognized"},
					},
				},
			})
			if err != nil {
				return nil, err
			}
			extractServiceInfo(commandResult.Extracted, info)
		}
		return info, nil

	case serviceTypeSystemctl:
		command = fmt.Sprintf("systemctl status %v ", info.Service)
	case serviceTypeStdService:
		command = fmt.Sprintf("service %v status", info.Service)
	case serviceTypeInitDaemon:
		command = fmt.Sprintf("%v status", info.Service)
	}

	commandResult, err := s.executeCommand(context, serviceType, target, &ManagedCommand{
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

	extractServiceInfo(commandResult.Extracted, info)
	return info, nil

}

func (s *daemonService) stopService(context *Context, request *DaemonStopRequest) (*DaemonInfo, error) {
	serviceInfo, err := s.checkService(context, &DaemonStatusRequest{
		Target:    request.Target,
		Service:   request.Service,
		Exclusion: request.Exclusion,
	})
	if err != nil {
		return nil, err
	}
	if !serviceInfo.IsActive() {
		return serviceInfo, nil
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
	commandResult, err := s.executeCommand(context, serviceInfo.Type, target, &ManagedCommand{
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
		return nil, fmt.Errorf("%v", commandResult.Stdout)
	}
	return s.checkService(context, &DaemonStatusRequest{
		Target:    request.Target,
		Service:   request.Service,
		Exclusion: request.Exclusion,
	})
}

func (s *daemonService) startService(context *Context, request *DaemonStartRequest) (*DaemonInfo, error) {
	serviceInfo, err := s.checkService(context, &DaemonStatusRequest{
		Target:    request.Target,
		Service:   request.Service,
		Exclusion: request.Exclusion,
	})
	if err != nil {
		return nil, err
	}

	if serviceInfo.IsActive() {
		return serviceInfo, nil
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
		if ! serviceInfo.Launched {
			command = fmt.Sprintf("launchctl load -F %v", serviceInfo.Path)
			_, err = s.executeCommand(context, serviceInfo.Type, target, &ManagedCommand{
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
		serviceInfo, _= s.checkService(context, &DaemonStatusRequest{
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

	commandResult, err := s.executeCommand(context, serviceInfo.Type, target, &ManagedCommand{
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
		return nil, fmt.Errorf("%v", commandResult.Stdout)
	}
	return s.checkService(context, &DaemonStatusRequest{
		Target:    request.Target,
		Service:   request.Service,
		Exclusion: request.Exclusion,
	})
}

//NewDaemonService creates a new system service.
func NewDaemonService() Service {
	var result = &daemonService{
		AbstractService: NewAbstractService(DaemonServiceID),
	}
	result.AbstractService.Service = result
	return result
}
