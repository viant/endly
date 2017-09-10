package endly

import (
	"fmt"
	"sync"
)

const TaskServiceId = "task"

type TaskStep struct {
	Name        string
	Service     string
	RequestName string
	Request     interface{}
}

type RunTaskRequest struct {
	Name         string
	Params map[string]interface{}
}

type RunTaskResponse struct {
	Name string
	Steps []*TaskStepInfo
}

type TaskStepInfo struct {
	Request interface{}
	Response interface{}
}


type Task struct {
	Name  string
	Steps []*TaskStep
}


type TaskService struct {
	*AbstractService
	registry map[string]*Task
}

func (s *TaskService) Register(task *Task) {
	s.registry[task.Name] = task
}

func (s *TaskService) Task(name string) (*Task, error) {
	if result, found:= s.registry[name];found {
		return result, nil
	}
	return nil, fmt.Errorf("Failed to lookup task: %v", name)
}


func (s *TaskService) runTask(context *Context, request *RunTaskRequest) (*RunTaskResponse, error) {

	task, err :=  s.Task(request.Name)
	if err != nil {
		return nil, err
	}
	var response = &RunTaskResponse{
		Steps:make([]*TaskStepInfo, 0),
	}
	var state = context.State()
	state.Apply(request.Params)

	for _, step := range task.Steps {

		service, err := context.Service(step.Service)
		if err != nil {
			return nil, err
		}
		serviceRequest, err := service.NewRequest(step.RequestName)
		if err != nil {
			return nil, err
		}
		err = converter.AssignConverted(serviceRequest, step.Request)
		if err != nil {
			return nil, err
		}
		serviceResponse := service.Run(context, serviceRequest)
		if serviceResponse.Error != nil {
			return nil, serviceResponse.Error
		}
		stepInfo := &TaskStepInfo{
			Request:  serviceRequest,
			Response: serviceResponse,
		}
		response.Steps = append(response.Steps, stepInfo)

	}
	return response, nil
}


func (s *TaskService) Run(context *Context, request interface{}) *Response {
	var response = &Response{Status: "ok"}

	switch actualRequest := request.(type) {
	case *RunTaskRequest:
		response.Response, response.Error = s.runTask(context, actualRequest)
	}
	if response.Error != nil {
		response.Status = "err"
	}
	return response
}

func (s *TaskService) NewRequest(name string) (interface{}, error) {
	switch name {
	case "run":
		return &RunTaskRequest{}, nil
	}
	return nil, fmt.Errorf("Unsupported name: %v", name)
}


var _taskService *TaskService
var _taskServiceMutex = &sync.Mutex{}

func GetTaskService() *TaskService {
	if _taskService != nil {
		return _taskService
	}
	_taskServiceMutex.Lock()
	defer _taskServiceMutex.Unlock()
	if _taskService != nil {
		return _taskService
	}
	var _taskService = &TaskService{
		AbstractService: NewAbstractService(TaskServiceId),
	}
	_taskService.AbstractService.Service = _taskService
	return _taskService
}
