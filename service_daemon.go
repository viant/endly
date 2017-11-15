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
	serviceTypeError = iota
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

func (s *daemonService) determineServiceType(context *Context, service, exclusion string, target *url.Resource) (int, string, error) {
	if exclusion != "" {
		exclusion = " | grep -v " + exclusion
	}
	commandResult, err := context.Execute(target, &ManagedCommand{
		Executions: []*Execution{
			{
				Command: fmt.Sprintf("ls /Library/LaunchDaemons/ | grep %v %v", service, exclusion),
			},
		},
	})
	if err != nil {
		return 0, "", err
	}
	if !CheckNoSuchFileOrDirectory(commandResult.Stdout()) {
		file := strings.TrimSpace(commandResult.Stdout())
		if len(file) > 0 {
			servicePath := path.Join("/Library/LaunchDaemons/", file)
			return serviceTypeLaunchCtl, servicePath, nil
		}
		return serviceTypeLaunchCtl, "", nil

	}

	commandResult, err = context.ExecuteAsSuperUser(target, &ManagedCommand{
		Options: &ExecutionOptions{
			Terminators: []string{"(END)"},
		},
		Executions: []*Execution{
			{
				Command: "service " + service + " status",
			},
			{
				Secure:      "",
				MatchOutput: "(END)", //quite multiline mode
				Command:     "Q",
			},
		},
	})

	if err != nil {
		return 0, "", err
	}
	if !CheckCommandNotFound(commandResult.Stdout()) {
		return serviceTypeStdService, service, nil
	}
	commandResult, err = context.ExecuteAsSuperUser(target, &ManagedCommand{
		Executions: []*Execution{
			{
				Command: "systemctl status " + service,
			},
		},
	})
	if err != nil {
		return 0, "", err
	}

	if !CheckCommandNotFound(commandResult.Stdout()) {
		return serviceTypeSystemctl, service, nil
	}

	return serviceTypeError, "", nil
}

func extractServiceInfo(state map[string]string, info *DaemonInfo) {
	if pid, ok := state["pid"]; ok {
		info.Pid = toolbox.AsInt(pid)
	}
	if value, ok := state["state"]; ok {
		state := vtclean.Clean(value, false)
		if strings.Contains(state, "inactive") {
			state = "not running"
		} else if strings.Contains(state, "active")   {
			state = "running"
		}
		info.State = state
	}
	if path, ok := state["path"]; ok {
		info.Path = path
	}
}

func (s *daemonService) checkService(context *Context, request *DaemonStatusRequest) (*DaemonInfo, error) {

	if request.Service == "" {
		return nil, fmt.Errorf("Service was empty")
	}
	serviceType, serviceInit, err := s.determineServiceType(context, request.Service, request.Exclusion, request.Target)
	if err != nil {
		return nil, err
	}

	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}

	var result = &DaemonInfo{
		Service: request.Service,
		Type:    serviceType,
		Init:    serviceInit,
	}
	command := ""

	if serviceInit == "" && serviceType == serviceTypeLaunchCtl {
		return result, nil
	}

	switch serviceType {
	case serviceTypeError:
		return nil, fmt.Errorf("Unknown daemon service type")
	case serviceTypeLaunchCtl:

		exclusion := request.Exclusion
		if exclusion != "" {
			exclusion = " | grep -v " + exclusion
		}

		commandResult, err := context.ExecuteAsSuperUser(target, &ManagedCommand{
			Executions: []*Execution{
				{
					Command: fmt.Sprintf("launchctl list | grep %v %v", request.Service, exclusion),
					Extraction: DataExtractions{
						{
							Key:     "pid",
							RegExpr: "(\\d+)[^\\d]+",
						},
					},
					Error: []string{"Unrecognized"},
				},
				{
					Command: "launchctl procinfo $pid",
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

		extractServiceInfo(commandResult.Extracted, result)
		return result, nil

	case serviceTypeSystemctl:
		command = fmt.Sprintf("systemctl status %v ", serviceInit)
	case serviceTypeStdService:
		command = fmt.Sprintf("service %v status", serviceInit)
	case serviceTypeInitDaemon:
		command = fmt.Sprintf("%v status", serviceInit)
	}

	commandResult, err := context.ExecuteAsSuperUser(target, &ManagedCommand{
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
				Secure:      "",
				MatchOutput: "(END)", //quite multiline mode
				Command:     "Q",
			},
		},
	})
	if err != nil {
		return nil, err
	}

	extractServiceInfo(commandResult.Extracted, result)
	return result, nil

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
		command = fmt.Sprintf("launchctl unload -F %v", serviceInfo.Init)
	case serviceTypeSystemctl:
		command = fmt.Sprintf("systemctl stop %v ", serviceInfo.Init)
	case serviceTypeStdService:
		command = fmt.Sprintf("service %v stop", serviceInfo.Init)
	case serviceTypeInitDaemon:
		command = fmt.Sprintf("%v stop", serviceInfo.Init)
	}

	commandResult, err := context.ExecuteAsSuperUser(target, &ManagedCommand{
		Executions: []*Execution{
			{
				Command: command,
			},
		},
	})
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
		command = fmt.Sprintf("launchctl load -F %v", serviceInfo.Init)
	case serviceTypeSystemctl:
		command = fmt.Sprintf("systemctl start %v ", serviceInfo.Init)
	case serviceTypeStdService:
		command = fmt.Sprintf("service %v start", serviceInfo.Init)
	case serviceTypeInitDaemon:
		command = fmt.Sprintf("%v start", serviceInfo.Init)
	}

	commandResult, err := context.ExecuteAsSuperUser(target, &ManagedCommand{
		Executions: []*Execution{
			{
				Command: command,
			},
		},
	})
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
