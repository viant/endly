package process

import (
	"fmt"
	"github.com/lunixbochs/vtclean"
	"github.com/viant/endly"
	"github.com/viant/endly/model/msg"
	"github.com/viant/endly/system/exec"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"
)

//ServiceID represents a system process service id
const ServiceID = "process"

type service struct {
	*endly.AbstractService
}

func (s *service) stopAllProcesses(context *endly.Context, request *StopRequest) (*StopResponse, error) {
	target := exec.GetServiceTarget(request.Target)

	status, err := s.checkProcess(context, &StatusRequest{
		Target:  target,
		Command: request.Input,
	})

	if err != nil {
		return nil, err
	}
	var response = &StopResponse{}
	for _, info := range status.Processes {
		commandResponse, err := s.stopProcess(context, &StopRequest{
			Target: target,
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

	actualCommand := request.Command
	if index := strings.Index(actualCommand, "grep "); index != -1 {
		actualCommand = string(actualCommand[index+5:])
	}

	for _, line := range strings.Split(runResponse.Stdout(), "\r\n") {
		line = vtclean.Clean(line, false)
		if strings.Contains(line, "grep") {
			continue
		}
		if !request.ExactCommand {
			index := strings.LastIndex(line, actualCommand)
			if index == -1 {
				continue
			}
			index += len(actualCommand)
			if index+1 < len(line) { //narrow grep result to command
				argsSeparator := string(line[index : index+1])
				if !(argsSeparator == " " || argsSeparator == "\t" || argsSeparator == "." || argsSeparator == "/") {
					continue
				}
			}
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

		if info.Pid == 0 {
			continue
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
	if request.Pid == 0 && request.Input != "" {
		return s.stopAllProcesses(context, request)
	}
	target := exec.GetServiceTarget(request.Target)
	var extractRequest = exec.NewExtractRequest(target, exec.DefaultOptions(), exec.NewExtractCommand(fmt.Sprintf("kill -9 %v", request.Pid), "", nil, nil))
	extractRequest.AutoSudo = true
	var runResponse = &exec.RunResponse{}
	if err := endly.Run(context, extractRequest, runResponse); err != nil {
		return nil, err
	}
	return &StopResponse{
		Stdout: runResponse.Stdout(),
	}, nil

}

func (s *service) stopExistingProcess(context *endly.Context, request *StartRequest) error {
	origProcesses, err := s.checkProcess(context, NewStatusRequest(request.Command, request.Target))
	if err != nil {
		return err
	}
	for _, process := range origProcesses.Processes {
		if strings.Join(process.Arguments, " ") == strings.Join(request.Arguments, " ") {
			if _, err := s.stopProcess(context, NewStopRequest(process.Pid, request.Target)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *service) buildStartProcessCommand(request *StartRequest) *exec.RunRequest {
	changeDirCommand := fmt.Sprintf("cd %v ", request.Directory)
	var startCommand = request.Command + " " + strings.Join(request.Arguments, " ") + " &"
	outputFile := path.Join(request.Directory, "nohup.out")
	var createNoHup = fmt.Sprintf("touch %v && chmod 666 %v", outputFile, outputFile)
	if request.ImmuneToHangups {
		toolbox.RemoveFileIfExist(outputFile)
		startCommand = fmt.Sprintf("nohup  %v", startCommand)
	}
	var runRequest = exec.NewRunRequest(request.Target, request.AsSuperUser, changeDirCommand, createNoHup, startCommand)
	if request.Options != nil {
		runRequest.Options = request.Options
	} else if runRequest.Options == nil {
		runRequest.Options = &exec.Options{}
	}
	runRequest.CheckError = true
	return runRequest
}

func (s *service) startProcess(context *endly.Context, request *StartRequest) (*StartResponse, error) {
	var response = &StartResponse{}
	err := s.stopExistingProcess(context, request)
	if err != nil {
		return nil, err
	}
	outputFile := path.Join(request.Directory, "nohup.out")
	startProcessRequest := s.buildStartProcessCommand(request)
	startProcessResponse := &exec.RunResponse{}
	if err = endly.Run(context, startProcessRequest, startProcessResponse); err != nil {
		return nil, err
	}
	response.Stdout = startProcessResponse.Output
	time.Sleep(time.Second)

	status, err := s.checkProcess(context, NewStatusRequest(request.Command, request.Target))
	if err != nil {
		return nil, err
	}
	response.Info = status.Processes
	response.Pid = status.Pid

	if request.ImmuneToHangups {
		stdout, err := s.readOutput(outputFile)
		if err == nil {
			response.Stdout += stdout
			context.Publish(msg.NewStdoutEvent("nohoup", stdout))
		}
		if request.Watch {
			go s.watchOutput(context, outputFile, len(stdout))
		}
	}

	return response, nil
}

func (s *service) watchOutput(context *endly.Context, location string, position int) {
	for !context.IsClosed() {
		stdout, err := s.readOutput(location)
		if err != nil {
			return
		}
		if position < len(stdout) {
			output := string(stdout[position:])
			context.Publish(msg.NewStdoutEvent("nohoup", output))
			position = len(stdout)
		}
		time.Sleep(time.Second)
	}
}

func (s *service) readOutput(location string) (string, error) {
	file, err := os.Open(location)
	if err != nil {
		return "", err
	}
	defer file.Close()
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *service) registerRoutes() {
	s.Register(&endly.Route{
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

	s.Register(&endly.Route{
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

	s.Register(&endly.Route{
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
