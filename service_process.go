package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"strings"
	"time"
)

//ProcessServiceID represents a system process service id
const ProcessServiceID = "process"

//ProcessServiceStartAction represents a process start action
const ProcessServiceStartAction = "start"

//ProcessServiceStatusAction represents a process status check
const ProcessServiceStatusAction = "status"

//ProcessServiceStopAction represents stop action
const ProcessServiceStopAction = "stop"

//ProcessServiceStopAllAction represents stop-all action
const ProcessServiceStopAllAction = "stop-all"

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
			response.Error = fmt.Sprintf("failed to start process: %v, %v", actualRequest.Command, err)
		}
	case *ProcessStopRequest:
		response.Response, err = s.stopProcess(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("failed to stop process: %v, %v", actualRequest.Pid, err)
		}
	case *ProcessStopAllRequest:
		response.Response, err = s.stopAllProcesses(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("failed to stop process: %v, %v", actualRequest.Input, err)
		}

	case *ProcessStatusRequest:
		response.Response, err = s.checkProcess(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("failed to stop process: %v, %v", actualRequest, err)
		}

	default:
		response.Error = fmt.Sprintf("unsupported request type: %T", request)

	}
	if response.Error != "" {
		response.Status = "err"
	}
	return response
}

func (s *processService) stopAllProcesses(context *Context, request *ProcessStopAllRequest) (*CommandResponse, error) {
	status, err := s.checkProcess(context, &ProcessStatusRequest{
		Target:  request.Target,
		Command: request.Input,
	})
	if err != nil {
		return nil, err
	}

	var respose *CommandResponse
	for _, info := range status.Processes {
		respose, err = s.stopProcess(context, &ProcessStopRequest{
			Target: request.Target,
			Pid:    info.Pid,
		})
		if err != nil {
			return nil, err
		}
	}
	return respose, nil
}

func (s *processService) checkProcess(context *Context, request *ProcessStatusRequest) (*ProcessStatusResponse, error) {
	var response = &ProcessStatusResponse{
		Processes: make([]*ProcessInfo, 0),
	}
	command := fmt.Sprintf("ps -ef | grep '%v'", request.Command)
	commandResponse, err := context.Execute(request.Target, &ExtractableCommand{
		Executions: []*Execution{
			{
				Command: command,
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
		line = strings.TrimSpace(line)
		columns, ok := ExtractColumns(line)
		if len(columns) < 3 || !ok {
			continue
		}
		info := &ProcessInfo{
			Pid:       toolbox.AsInt(columns[1]),
			Command:   request.Command,
			Arguments: make([]string, 0),
			Stdin:     command,
			Stdout:    line,
		}
		var expectArgument = false
		for _, column := range columns {
			if expectArgument {
				info.Arguments = append(info.Arguments, column)
				continue
			}
			if strings.Contains(column, request.Command) {
				info.Name = column
				expectArgument = true
			}
		}
		info.Stdout = strings.Join(columns, " ")
		response.Processes = append(response.Processes, info)
	}
	if len(response.Processes) > 0 {
		response.Pid = response.Processes[0].Pid
	}
	return response, nil
}

func (s *processService) stopProcess(context *Context, request *ProcessStopRequest) (*CommandResponse, error) {
	commandResult, err := context.ExecuteAsSuperUser(request.Target, &ExtractableCommand{
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
		Target:  request.Target,
		Command: request.Command,
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

	var startCommand = request.Command + " " + strings.Join(request.Arguments, " ") + " &"
	if request.ImmuneToHangups {
		startCommand = fmt.Sprintf("nohup  %v", startCommand)
	}
	_, err = context.Execute(request.Target, &ExtractableCommand{
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
	case ProcessServiceStartAction:
		return &ProcessStartRequest{}, nil
	case ProcessServiceStatusAction:
		return &ProcessStatusRequest{}, nil
	case ProcessServiceStopAction:
		return &ProcessStopRequest{}, nil
	case ProcessServiceStopAllAction:
		return &ProcessStopAllRequest{}, nil

	}
	return s.AbstractService.NewRequest(action)
}

//NewProcessService returns a new system process service.
func NewProcessService() Service {
	var result = &processService{
		AbstractService: NewAbstractService(ProcessServiceID,
			ProcessServiceStartAction,
			ProcessServiceStatusAction,
			ProcessServiceStopAction,
			ProcessServiceStopAllAction),
	}
	result.AbstractService.Service = result
	return result
}
