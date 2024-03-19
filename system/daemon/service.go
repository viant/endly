package daemon

import (
	"fmt"
	"github.com/lunixbochs/vtclean"
	"github.com/viant/endly"
	"github.com/viant/endly/internal/util"
	"github.com/viant/endly/model"
	"github.com/viant/endly/model/location"
	"github.com/viant/endly/system/exec"
	"github.com/viant/toolbox"
	"path"
	"strings"
)

// ServiceID represents system daemon service
const ServiceID = "daemon"

const (
	serviceTypeError = iota
	serviceTypeInitDaemon
	serviceTypeLaunchCtl
	serviceTypeStdService
	serviceTypeSystemctl
)

type service struct {
	*endly.AbstractService
}

func (s *service) getDarwinLaunchServiceInfo(context *endly.Context, target *location.Resource, request *StatusRequest, info *Info) error {

	if request.Exclusion != "" {
		request.Exclusion = " | grep -v " + request.Exclusion
	}

	runResponse := &exec.RunResponse{}
	runRequest := exec.NewExtractRequest(target, exec.DefaultOptions(), exec.NewExtractCommand(fmt.Sprintf("ls /Library/LaunchDaemons/ | grep %v %v", request.Service, request.Exclusion), "", nil, nil))
	if err := endly.Run(context, runRequest, runResponse); err != nil {
		return err
	}

	file := strings.TrimSpace(runResponse.Stdout())
	if len(file) > 0 {
		info.Path = path.Join("/Library/LaunchDaemons/", file)
	}

	runRequest = exec.NewExtractRequest(target, exec.DefaultOptions(), exec.NewExtractCommand(fmt.Sprintf("launchctl list | grep %v %v", request.Service, request.Exclusion), "", nil, nil))
	if err := endly.Run(context, runRequest, runResponse); err != nil {
		return err
	}
	stdout := runResponse.Stdout()
	for _, line := range strings.Split(stdout, "\n") {
		columns, ok := util.ExtractColumns(line)
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

func (s *service) determineServiceType(context *endly.Context, service, exclusion string, target *location.Resource) (int, error) {
	session, err := exec.TerminalSession(context, target)
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

	for _, candidate := range systemTypeCommands {

		var runResponse = &exec.RunResponse{}
		var extractRequest = exec.NewExtractRequest(target, exec.DefaultOptions(), exec.NewExtractCommand(candidate.command, "", nil, nil))
		if err = endly.Run(context, extractRequest, runResponse); err != nil {
			break
		}
		var stdout = runResponse.Stdout()
		if !util.CheckNoSuchFileOrDirectory(stdout) && !util.CheckCommandNotFound(stdout) {
			session.DaemonType = candidate.systemType
			return session.DaemonType, nil
		}
	}
	return serviceTypeError, nil
}

func extractServiceInfo(stdout string, state map[string]interface{}, info *Info) {
	if pid, ok := state["pid"]; ok {
		info.Pid = toolbox.AsInt(pid)
	}
	if val, ok := state["state"]; ok {
		value := val.(string)
		state := vtclean.Clean(value, false)
		if strings.Contains(state, "inactive") {
			state = "not running"
		} else if strings.Contains(state, "active") {
			state = "running"
		}
		info.State = state
	}
	if val, ok := state["path"]; ok {
		info.Path = val.(string)
	}

	//deal with service deamon		info.Pid = docker start/running, process 48628
	if info.State == "" {
		candidate := vtclean.Clean(stdout, false)
		if strings.Contains(candidate, "start/running") {
			info.State = "running"
			if columns, ok := util.ExtractColumns(info.State); ok {
				if len(columns) > 0 {
					info.Pid = toolbox.AsInt(columns[len(columns)-1])
				}
			}
		} else if strings.Contains(candidate, "stop/waiting") {
			info.State = "not running"
		}
	}
}

func (s *service) executeCommand(context *endly.Context, serviceType int, target *location.Resource, request *exec.ExtractRequest) (*exec.RunResponse, error) {
	var runResponse = &exec.RunResponse{}
	if serviceType != serviceTypeLaunchCtl {
		request.SuperUser = true
	}
	return runResponse, endly.Run(context, request, runResponse)
}

func (s *service) isLaunchCtlDomainMissing(info *Info) bool {
	return (info.Path == "" || info.Domain == "") && info.Type == serviceTypeLaunchCtl
}

func (s *service) determineCheckCommand(context *endly.Context, target *location.Resource, serviceType int, info *Info) (command string, err error) {
	switch serviceType {
	case serviceTypeError:
		return "", fmt.Errorf("unknown daemon service type")

	case serviceTypeLaunchCtl:

		if info.Pid > 0 {
			var runResponse = &exec.RunResponse{}
			var extractRequest = exec.NewExtractRequest(target, exec.DefaultOptions(), exec.NewExtractCommand(fmt.Sprintf("launchctl procinfo %v", info.Pid), "", nil, []string{"Unrecognized"},
				model.NewExtract("path", "program path[\\s|\\t]+=[\\s|\\t]+([^\\s]+)", false, false),
				model.NewExtract("state", "state = (running)", false, false)))
			extractRequest.SuperUser = true

			err = endly.Run(context, extractRequest, runResponse)
			if err != nil {
				return "", err
			}
			extractServiceInfo(runResponse.Stdout(), runResponse.Data, info)
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

func (s *service) checkService(context *endly.Context, request *StatusRequest) (*Info, error) {
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
	var info = &Info{
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

	var options = exec.DefaultOptions()
	options.Terminators = append(options.Terminators, "(END)")

	commandResult, err := s.executeCommand(context, serviceType, target, exec.NewExtractRequest(
		target, options, exec.NewExtractCommand(command, "", nil, nil,
			model.NewExtract("pid", "[^└]+└─(\\d+).+", true, false),
			model.NewExtract("pid", "Main PID: (\\d+).+", false, false),
			model.NewExtract("state", "[\\s|\\t]+Active:\\s+(\\S+)", false, false),
			model.NewExtract("path", "[^└]+└─\\d+[\\s\\t].(.+)", false, false)),
		exec.NewExtractCommand("Q", "(END)", nil, nil)))

	if err != nil {
		return nil, err
	}

	extractServiceInfo(commandResult.Stdout(), commandResult.Data, info)
	return info, nil

}

func (s *service) stopService(context *endly.Context, request *StopRequest) (*StopResponse, error) {
	serviceInfo, err := s.checkService(context, &StatusRequest{
		Target:    request.Target,
		Service:   request.Service,
		Exclusion: request.Exclusion,
	})
	if err != nil {
		return nil, err
	}
	if !serviceInfo.IsActive() {
		return &StopResponse{serviceInfo}, nil
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
	commandResult, err := s.executeCommand(context, serviceInfo.Type, target,
		exec.NewExtractRequest(target, exec.DefaultOptions(), exec.NewExtractCommand(command, "", nil, nil)))
	if err != nil {
		return nil, err
	}
	if util.CheckCommandNotFound(commandResult.Stdout()) {
		return nil, fmt.Errorf("%v", commandResult.Stdout())
	}
	info, err := s.checkService(context, &StatusRequest{
		Target:    request.Target,
		Service:   request.Service,
		Exclusion: request.Exclusion,
	})
	return &StopResponse{info}, err
}

func (s *service) startService(context *endly.Context, request *StartRequest) (*StartResponse, error) {
	serviceInfo, err := s.checkService(context, &StatusRequest{
		Target:    request.Target,
		Service:   request.Service,
		Exclusion: request.Exclusion,
	})
	if err != nil {
		return nil, err
	}
	if serviceInfo.IsActive() {
		return &StartResponse{Info: serviceInfo}, nil
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
			_, err = s.executeCommand(context, serviceInfo.Type, target,
				exec.NewExtractRequest(target, exec.DefaultOptions(), exec.NewExtractCommand(command, "", nil, nil)))
			if err != nil {
				return nil, err
			}
		}
		serviceInfo, _ = s.checkService(context, &StatusRequest{
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
	commandResult, err := s.executeCommand(context, serviceInfo.Type, target,
		exec.NewExtractRequest(target, exec.DefaultOptions(), exec.NewExtractCommand(command, "", nil, nil)))
	if err != nil {
		return nil, err
	}
	if util.CheckCommandNotFound(commandResult.Stdout()) {
		return nil, fmt.Errorf("%v", commandResult.Stdout())
	}

	serviceInfo, err = s.checkService(context, &StatusRequest{
		Target:    request.Target,
		Service:   request.Service,
		Exclusion: request.Exclusion,
	})
	if serviceInfo != nil && !serviceInfo.IsActive() {
		return nil, fmt.Errorf("%v service is inactive", request.Service)
	}
	return &StartResponse{Info: serviceInfo}, err
}

func (s *service) registerRoutes() {
	s.Register(&endly.Route{
		Action: "start",
		RequestInfo: &endly.ActionInfo{
			Description: "start provided service on target host",
		},
		RequestProvider: func() interface{} {
			return &StartRequest{}
		},
		ResponseProvider: func() interface{} {
			return &StartResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*StartRequest); ok {
				return s.startService(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
	s.Register(&endly.Route{
		Action: "status",
		RequestInfo: &endly.ActionInfo{
			Description: "check service status on target host",
		},
		RequestProvider: func() interface{} {
			return &StatusRequest{}
		},
		ResponseProvider: func() interface{} {
			return &Info{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*StatusRequest); ok {
				return s.checkService(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
	s.Register(&endly.Route{
		Action: "stop",
		RequestInfo: &endly.ActionInfo{
			Description: "stop service on target host",
		},
		RequestProvider: func() interface{} {
			return &StopRequest{}
		},
		ResponseProvider: func() interface{} {
			return &Info{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*StopRequest); ok {
				return s.stopService(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

}

// NewDaemonService creates a new system service.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.registerRoutes()
	result.AbstractService.Service = result
	return result
}
