package process

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/system/exec"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
	"strings"
	"time"
)

//ServiceID represents a system process service id
const ServiceID = "process"

type service struct {
	*endly.AbstractService
}

func (s *service) stopAllProcesses(context *endly.Context, request *StopAllRequest) (*StopAllResponse, error) {
	status, err := s.checkProcess(context, &StatusRequest{
		Target:  request.Target,
		Command: request.Input,
	})
	if err != nil {
		return nil, err
	}
	var response = &StopAllResponse{}
	for _, info := range status.Processes {
		commandResponse, err := s.stopProcess(context, &StopRequest{
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

func (s *service) checkProcess(context *endly.Context, request *StatusRequest) (*StatusResponse, error) {
	var response = &StatusResponse{
		Processes: make([]*Info, 0),
	}

	command := fmt.Sprintf("ps -ef | grep %v", request.Command)
	if strings.Contains(request.Command, " ") && !strings.Contains(request.Command, "|") {
		command = fmt.Sprintf("ps -ef | grep '%v'", request.Command)
	}

	var extractRequest = exec.NewExtractRequest(request.Target, exec.DefaultOptions(), exec.NewExtractCommand(command, "", nil, nil))
	var runResponse = &exec.RunResponse{}
	if err := endly.Run(context, extractRequest, runResponse); err != nil {
		return nil, err
	}

	for _, line := range strings.Split(runResponse.Stdout(), "\r\n") {
		if strings.Contains(line, "grep") {
			continue
		}
		line = strings.TrimSpace(line)
		columns, ok := util.ExtractColumns(line)
		if len(columns) < 3 || !ok {
			continue
		}
		info := &Info{
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

func (s *service) stopProcess(context *endly.Context, request *StopRequest) (*StopResponse, error) {
	var extractRequest = exec.NewExtractRequest(request.Target, exec.DefaultOptions(), exec.NewExtractCommand(fmt.Sprintf("kill -9 %v", request.Pid), "", nil, nil))
	extractRequest.SuperUser = true
	var runResponse = &exec.RunResponse{}
	if err := endly.Run(context, extractRequest, runResponse); err != nil {
		return nil, err
	}
	return &StopResponse{
		Stdout: runResponse.Stdout(),
	}, nil

}

func indexProcesses(processes ...*Info) map[int]*Info {
	var result = make(map[int]*Info)
	for _, process := range processes {
		result[process.Pid] = process
	}
	return result
}

func (s *service) startProcess(context *endly.Context, request *StartRequest) (*StartResponse, error) {
	origProcesses, err := s.checkProcess(context, &StatusRequest{
		Target:  request.Target,
		Command: request.Command,
	})

	var result = &StartResponse{}
	if err != nil {
		return nil, err
	}
	for _, process := range origProcesses.Processes {
		if strings.Join(process.Arguments, " ") == strings.Join(request.Arguments, " ") {
			_, err := s.stopProcess(context, &StopRequest{
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
	if err = endly.Run(context, exec.NewRunRequest(request.Target, request.AsSuperUser, changeDirCommand, startCommand), nil); err != nil {
		return nil, err
	}
	time.Sleep(time.Second)
	newProcesses, err := s.checkProcess(context, &StatusRequest{
		Target:  request.Target,
		Command: request.Command,
	})
	if err != nil {
		return nil, err
	}

	result.Info = make([]*Info, 0)
	existingProcesses := indexProcesses(origProcesses.Processes...)

	for _, candidate := range newProcesses.Processes {
		if _, has := existingProcesses[candidate.Pid]; !has {
			result.Info = append(result.Info, candidate)
			break
		}
	}
	return result, nil
}

func (s *service) registerRoutes() {
	s.Register(&endly.ServiceActionRoute{
		Action: "start",
		RequestInfo: &endly.ActionInfo{
			Description: "start process",
		},
		RequestProvider: func() interface{} {
			return &StartRequest{}
		},
		ResponseProvider: func() interface{} {
			return &StartResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*StartRequest); ok {
				return s.startProcess(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "stop",
		RequestInfo: &endly.ActionInfo{
			Description: "stop process",
		},
		RequestProvider: func() interface{} {
			return &StopRequest{}
		},
		ResponseProvider: func() interface{} {
			return &StopResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*StopRequest); ok {
				return s.stopProcess(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "status",
		RequestInfo: &endly.ActionInfo{
			Description: "check process status",
		},
		RequestProvider: func() interface{} {
			return &StatusRequest{}
		},
		ResponseProvider: func() interface{} {
			return &StatusResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*StatusRequest); ok {
				return s.checkProcess(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "stop-all",
		RequestInfo: &endly.ActionInfo{
			Description: "stop all matching processes",
		},
		RequestProvider: func() interface{} {
			return &StopAllRequest{}
		},
		ResponseProvider: func() interface{} {
			return &StopAllResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*StopAllRequest); ok {
				return s.stopAllProcesses(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//New creates new system process service.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
