package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"strings"
	"time"
)

const ProcessServiceId = "process"

type StartProcessRequest struct {
	Name          string
	Target        *Resource
	Options       *ExecutionOptions
	SystemService bool
	Directory     string
	Command       string
	Arguments     []string
}

type ProcessInfo struct {
	Name      string
	Pid       int
	Command   string
	Arguments []string
}

type StatusProcessRequest struct {
	Target  *Resource
	Command string
}

type StopProcessRequest struct {
	Target *Resource
	Pid    int
}

type processService struct {
	*AbstractService
}

func (s *processService) Run(context *Context, request interface{}) *Response {
	var response = &Response{Status: "ok"}

	switch actualRequest := request.(type) {
	case *StartProcessRequest:
		response.Response, response.Error = s.startProcess(context, actualRequest)
	}
	if response.Error != nil {
		response.Status = "err"
	}
	return response
}

func (s *processService) checkProcess(context *Context, request *StatusProcessRequest) ([]*ProcessInfo, error) {
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
	var result = make([]*ProcessInfo, 0)
	for _, line := range strings.Split(commandResponse.Stdout[0], "\r\n") {
		if strings.Contains(line, "grep") {
			continue
		}
		spaceIndex := strings.Index(line, " ")
		if spaceIndex < 2 {
			continue
		}
		argumentsIndex := strings.Index(line, request.Command)
		var arguments []string
		if argumentsIndex != -1 {
			args := strings.Trim(line[argumentsIndex+len(request.Command)+1:], " &\t")
			arguments = strings.Split(args, " ")
		}
		info := &ProcessInfo{
			Pid:       toolbox.AsInt(string(line[:spaceIndex])),
			Command:   request.Command,
			Arguments: arguments,
		}
		result = append(result, info)
	}
	return result, nil
}

func (s *processService) stopProcess(context *Context, request *StopProcessRequest) (*CommandResult, error) {
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
	fmt.Printf("kill %v\n", commandResult.Stdout)
	return commandResult, err
}

func indexProcesses(processes ...*ProcessInfo) map[int]*ProcessInfo {
	var result = make(map[int]*ProcessInfo)
	for _, process := range processes {
		result[process.Pid] = process
	}
	return result
}

func (s *processService) startProcess(context *Context, request *StartProcessRequest) (*ProcessInfo, error) {
	origProcesses, err := s.checkProcess(context, &StatusProcessRequest{
		Target:  request.Target,
		Command: request.Command,
	})
	if err != nil {
		return nil, err
	}
	for _, process := range origProcesses {

		if strings.Join(process.Arguments, " ") == strings.Join(request.Arguments, " ") {
			_, err := s.stopProcess(context, &StopProcessRequest{
				Pid:    process.Pid,
				Target: request.Target,
			})
			if err != nil {
				return nil, err
			}
		}
	}
	_, err = context.Execute(request.Target, &ManagedCommand{
		Options: request.Options,
		Executions: []*Execution{
			{
				Command: fmt.Sprintf("cd %v ", request.Directory),
			},
			{
				Command: fmt.Sprintf("nohup %v %v &", request.Command, strings.Join(request.Arguments, " ")),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	time.Sleep(time.Second)
	newProcesses, err := s.checkProcess(context, &StatusProcessRequest{
		Target:  request.Target,
		Command: request.Command,
	})
	if err != nil {
		return nil, err
	}

	if len(newProcesses) == 1 {
		return newProcesses[0], nil
	}
	existingProcesses := indexProcesses(origProcesses...)
	var result *ProcessInfo
	for _, candidate := range newProcesses {

		if _, has := existingProcesses[candidate.Pid]; !has {
			result = candidate
			break
		}
	}
	if result == nil {
		return nil, fmt.Errorf("Failed to get info about prorcess %v", request.Command)
	}
	return result, nil
}

func (s *processService) NewRequest(name string) (interface{}, error) {
	switch name {
	case "start":
		return &StartProcessRequest{}, nil
	case "check":
		return &StatusProcessRequest{}, nil
	case "stop":
		return &StopProcessRequest{}, nil

	}
	return nil, fmt.Errorf("Unsupported name: %v", name)
}

func NewProcessService() Service {
	var result = &processService{
		AbstractService: NewAbstractService(ProcessServiceId),
	}
	result.AbstractService.Service = result
	return result
}
