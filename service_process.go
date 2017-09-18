package endly

import (
	"fmt"
	"github.com/viant/endly/common"
	"github.com/viant/toolbox"
	"strings"
	"time"
)

const ProcessServiceId = "process"
const processesKey = "pid"

type ProcessStartRequest struct {
	Name          string
	Target        *Resource
	Options       *ExecutionOptions
	SystemService bool
	Directory     string
	Command       string
	Arguments     []string
}

type ProcessStartResponse struct {
	Command string
	Info []*ProcessInfo
}

type ProcessInfo struct {
	Name      string
	Pid       int
	Command   string
	Arguments []string
	Stdin     string
	Stdout    string
}

type ProcessStatusRequest struct {
	Target  *Resource
	Command string
}

type ProcessStopRequest struct {
	Target *Resource
	Pid    int
}

type processService struct {
	*AbstractService
}

func (s *processService) Run(context *Context, request interface{}) *ServiceResponse {
	var response = &ServiceResponse{Status: "ok"}
	var err error
	switch actualRequest := request.(type) {
	case *ProcessStartRequest:
		response.Response, err = s.startProcess(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to start process: %v, %v", actualRequest.Name, err)
		}
	case *ProcessStopRequest:
		response.Response, err = s.stopProcess(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to stop process: %v, %v", actualRequest.Pid, err)
		}

	}
	if response.Error != "" {
		response.Status = "err"
	}
	return response
}

func (s *processService) checkProcess(context *Context, request *ProcessStatusRequest) ([]*ProcessInfo, error) {
	commandResponse, err := context.Execute(request.Target, &ManagedCommand{
		Executions: []*Execution{
			{
				Command: "ps -ev | grep " + request.Command,
			},
		},
	})

	if err != nil {
		return nil, err
	}

	var state = context.State()
	if !state.Has(processesKey) {
		state.Put(processesKey, common.NewMap())
	}
	var processes = state.GetMap(processesKey)
	processes.Put(request.Command, 0)
	var result = make([]*ProcessInfo, 0)

	for _, line := range strings.Split(commandResponse.Stdout(), "\r\n") {
		if strings.Contains(line, "grep") {
			continue
		}
		pid, ok := ExtractColumn(line, 0)
		if !ok {
			continue
		}
		argumentsIndex := strings.Index(line, request.Command)
		var arguments []string
		if argumentsIndex != -1 {
			args := strings.Trim(line[argumentsIndex+len(request.Command)+1:], " &\t")
			arguments = strings.Split(args, " ")
		}
		info := &ProcessInfo{
			Pid:       toolbox.AsInt(pid),
			Command:   request.Command,
			Arguments: arguments,
			Stdout:    line,
		}
		result = append(result, info)
		processes.Put(request.Command, info.Pid)
	}
	return result, nil
}

func (s *processService) stopProcess(context *Context, request *ProcessStopRequest) (*CommandInfo, error) {
	commandResult, err := context.Execute(request.Target, &ManagedCommand{
		Executions: []*Execution{
			{
				Command: fmt.Sprintf("kill -9 %v", request.Pid),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	state := context.State()
	if !state.Has(processesKey) {
		state.Put(processesKey, common.NewMap())
	}
	var processes = state.GetMap(processesKey)
	for k, pid := range processes {
		if toolbox.AsInt(pid) == request.Pid {
			state[k] = 0
			break
		}
	}
	return commandResult, err
}

func indexProcesses(processes ...*ProcessInfo) map[int]*ProcessInfo {
	var result = make(map[int]*ProcessInfo)
	for _, process := range processes {
		result[process.Pid] = process
	}
	return result
}

func (s *processService) startProcess(context *Context, request *ProcessStartRequest) (*ProcessStartResponse, error) {
	origProcesses, err := s.checkProcess(context, &ProcessStatusRequest{
		Target:  request.Target,
		Command: request.Command,
	})

	var result = &ProcessStartResponse{}
	if err != nil {
		return nil, err
	}
	for _, process := range origProcesses {
		if strings.Join(process.Arguments, " ") == strings.Join(request.Arguments, " ") {
			_, err := s.stopProcess(context, &ProcessStopRequest{
				Pid:    process.Pid,
				Target: request.Target,
			})
			if err != nil {
				return nil, err
			}
		}
	}
	changeDirCommand := fmt.Sprintf("cd %v ", request.Directory)
	startCommand := fmt.Sprintf("nohup %v %v &", request.Command, strings.Join(request.Arguments, " "))
	 _, err = context.Execute(request.Target, &ManagedCommand{
		Options: request.Options,
		Executions: []*Execution{
			{
				Command: changeDirCommand,
			},
			{
				Command: startCommand,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	time.Sleep(time.Second)
	newProcesses, err := s.checkProcess(context, &ProcessStatusRequest{
		Target:  request.Target,
		Command: request.Command,
	})
	if err != nil {
		return nil, err
	}

	result.Info = make([]*ProcessInfo, 0)
	existingProcesses := indexProcesses(origProcesses...)

	for _, candidate := range newProcesses {
		if _, has := existingProcesses[candidate.Pid]; !has {
			result.Info = append(result.Info, candidate)
			break
		}
	}
	return result, nil
}


func (s *processService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "start":
		return &ProcessStartRequest{}, nil
	case "check":
		return &ProcessStatusRequest{}, nil
	case "stop":
		return &ProcessStopRequest{}, nil

	}
	return s.AbstractService.NewRequest(action)
}

func NewProcessService() Service {
	var result = &processService{
		AbstractService: NewAbstractService(ProcessServiceId),
	}
	result.AbstractService.Service = result
	return result
}
