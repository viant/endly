package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"strings"
	"time"
)

//ProcessServiceID represents a system process service id
const ProcessServiceID = "process"

type processService struct {
	*AbstractService
}

func (s *processService) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok"}
	defer s.End(context)(startEvent, Pairs("response", response))
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
	case *ProcessStatusRequest:
		response.Response, err = s.checkProcess(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to stop process: %v, %v", actualRequest, err)
		}

	default:
		response.Error = fmt.Sprintf("Unsupported request type: %T", request)

	}
	if response.Error != "" {
		response.Status = "err"
	}
	return response
}

func (s *processService) checkProcess(context *Context, request *ProcessStatusRequest) (*ProcessStatusResponse, error) {
	var response = &ProcessStatusResponse{
		Processes: make([]*ProcessInfo, 0),
	}
	commandResponse, err := context.Execute(request.Target, &ManagedCommand{
		Executions: []*Execution{
			{
				Command: "ps -ev | grep " + request.Name,
			},
		},
	})

	if err != nil {
		return nil, err
	}

	for _, line := range strings.Split(commandResponse.Stdout(), "\r\n") {
		if strings.Contains(line, "grep") {
			continue
		}
		pid, ok := ExtractColumn(line, 0)
		if !ok {
			continue
		}
		argumentsIndex := strings.Index(line, request.Name)
		var arguments []string
		if argumentsIndex != -1 {
			args := strings.Trim(line[argumentsIndex+len(request.Name)+1:], " &\t")
			arguments = strings.Split(args, " ")
		}
		info := &ProcessInfo{
			Pid:       toolbox.AsInt(pid),
			Command:   request.Name,
			Arguments: arguments,
			Stdout:    line,
		}
		response.Processes = append(response.Processes, info)
	}
	if len(response.Processes) > 0 {
		response.Pid = response.Processes[0].Pid
	}
	return response, nil
}

func (s *processService) stopProcess(context *Context, request *ProcessStopRequest) (*CommandResponse, error) {
	commandResult, err := context.ExecuteAsSuperUser(request.Target, &ManagedCommand{
		Executions: []*Execution{
			{
				Command: fmt.Sprintf("kill -9 %v", request.Pid),
			},
		},
	})
	if err != nil {
		return nil, err
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
		Target: request.Target,
		Name:   request.Name,
	})

	var result = &ProcessStartResponse{}
	if err != nil {
		return nil, err
	}
	for _, process := range origProcesses.Processes {
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
	startCommand := fmt.Sprintf("nohup %v %v &", request.Name, strings.Join(request.Arguments, " "))
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
		Target: request.Target,
		Name:   request.Name,
	})
	if err != nil {
		return nil, err
	}

	result.Info = make([]*ProcessInfo, 0)
	existingProcesses := indexProcesses(origProcesses.Processes...)

	for _, candidate := range newProcesses.Processes {
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

//NewProcessService returns a new system process service.
func NewProcessService() Service {
	var result = &processService{
		AbstractService: NewAbstractService(ProcessServiceID),
	}
	result.AbstractService.Service = result
	return result
}
