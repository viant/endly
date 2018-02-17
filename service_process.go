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

func (s *processService) stopAllProcesses(context *Context, request *ProcessStopAllRequest) (*ProcessStopAllResponse, error) {
	status, err := s.checkProcess(context, &ProcessStatusRequest{
		Target:  request.Target,
		Command: request.Input,
	})
	if err != nil {
		return nil, err
	}
	var response = &ProcessStopAllResponse{}
	for _, info := range status.Processes {
		commandResponse, err := s.stopProcess(context, &ProcessStopRequest{
			Target: request.Target,
			Pid:    info.Pid,
		})
		if err != nil {
			return nil, err
		}
		if len(response.Stdout) > 0 {
			response.Stdout += "\n"
		}
		response.Stdout += commandResponse.Stdout
	}
	return response, nil
}

func (s *processService) checkProcess(context *Context, request *ProcessStatusRequest) (*ProcessStatusResponse, error) {
	var response = &ProcessStatusResponse{
		Processes: make([]*ProcessInfo, 0),
	}

	command := fmt.Sprintf("ps -ef | grep %v", request.Command)
	if strings.Contains(request.Command, " ") && !strings.Contains(request.Command, "|") {
		command = fmt.Sprintf("ps -ef | grep '%v'", request.Command)
	}
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

func (s *processService) stopProcess(context *Context, request *ProcessStopRequest) (*ProcessStopResponse, error) {
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
	return &ProcessStopResponse{
		Stdout: commandResult.Stdout(),
	}, nil

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

func (s *processService) registerRoutes() {
	s.Register(&ServiceActionRoute{
		Action: "start",
		RequestInfo: &ActionInfo{
			Description: "start process",
		},
		RequestProvider: func() interface{} {
			return &ProcessStartRequest{}
		},
		ResponseProvider: func() interface{} {
			return &ProcessStartResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*ProcessStartRequest); ok {
				return s.startProcess(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "stop",
		RequestInfo: &ActionInfo{
			Description: "stop process",
		},
		RequestProvider: func() interface{} {
			return &ProcessStopRequest{}
		},
		ResponseProvider: func() interface{} {
			return &ProcessStopResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*ProcessStopRequest); ok {
				return s.stopProcess(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "status",
		RequestInfo: &ActionInfo{
			Description: "check process status",
		},
		RequestProvider: func() interface{} {
			return &ProcessStatusRequest{}
		},
		ResponseProvider: func() interface{} {
			return &ProcessStatusResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*ProcessStatusRequest); ok {
				return s.checkProcess(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "stop-all",
		RequestInfo: &ActionInfo{
			Description: "stop all matching processes",
		},
		RequestProvider: func() interface{} {
			return &ProcessStopAllRequest{}
		},
		ResponseProvider: func() interface{} {
			return &ProcessStopAllResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*ProcessStopAllRequest); ok {
				return s.stopAllProcesses(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//NewProcessService returns a new system process service.
func NewProcessService() Service {
	var result = &processService{
		AbstractService: NewAbstractService(ProcessServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
