package endly

import (
	"errors"
	"fmt"
	"github.com/viant/endly/common"
	"github.com/viant/toolbox"
	"strings"
	"time"
)

const WorkflowServiceId = "workflow"

type WorkflowRunRequest struct {
	WorkflowURL string
	Name        string
	Params      map[string]interface{}
	Tasks       map[string]string
	Task        string
}

type WorkflowRunResponse struct {
	Name            string
	Data            map[string]interface{}
	SessionInfo     *SessionInfo
	TasksActivities []*WorkflowTaskActivity
}

type WorkflowTaskActivity struct {
	Task              string
	ServiceActivities []*WorkflowServiceActivity
	Data              map[string]interface{}
	Skipped           string
}

type WorkflowServiceActivity struct {
	Service         string
	Action          string
	ServiceRequest  interface{}
	ServiceResponse interface{}
	Error           string
	Skipped         string
}

type WorkflowRegisterRequest struct {
	Workflow *Workflow
}

type WorkflowLoadRequest struct {
	Source *Resource
}

type WorkflowService struct {
	*AbstractService
	dao      *WorkflowDao
	registry map[string]*Workflow
}

func (s *WorkflowService) Register(workflow *Workflow) error {
	err := workflow.Validate()
	if err != nil {
		return err
	}
	s.registry[workflow.Name] = workflow
	return nil
}

func (s *WorkflowService) Workflow(name string) (*Workflow, error) {
	if result, found := s.registry[name]; found {
		return result, nil
	}
	return nil, fmt.Errorf("Failed to lookup workflow: %v", name)
}

func (s *WorkflowService) evaluateRunCriteria(context *Context, criteria string) (bool, error) {

	if criteria == "" {
		return true, nil
	}

	colonPosition := strings.Index(criteria, ":")
	if colonPosition == -1 {
		return true, nil
	}
	fragments := strings.Split(criteria, ":")
	actualOperand := context.Expand(strings.TrimSpace(fragments[0]))
	expectedOperand := context.Expand(strings.TrimSpace(fragments[1]))
	validator := &Validator{}

	var result, err = validator.Check(expectedOperand, actualOperand)
	return result, err
}

func isTaskAllowed(candidate *WorkflowTask, request *WorkflowRunRequest) (bool, map[int]bool) {

	if (len(request.Tasks) == 0 && request.Task == "") || request.Task == candidate.Name {
		return true, map[int]bool{}
	}

	if allowedActions, has := request.Tasks[candidate.Name]; has {
		var allowedActionIndexes = make(map[int]bool)
		for _, index := range strings.Split(allowedActions, ",") {
			if index == "" {
				continue
			}
			allowedActionIndexes[toolbox.AsInt(index)] = true
		}
		return true, allowedActionIndexes
	}
	return false, nil
}

func (s *WorkflowService) runWorkflow(context *Context, request *WorkflowRunRequest) (*WorkflowRunResponse, error) {
	workflow, err := s.Workflow(request.Name)
	if err != nil {
		return nil, err
	}

	var state = context.State()
	var response = &WorkflowRunResponse{
		TasksActivities: make([]*WorkflowTaskActivity, 0),
		Data:            make(map[string]interface{}),
	}
	previousWorkflow := context.Workflow()
	if previousWorkflow != nil {
		defer context.Put(workflowKey, previousWorkflow)
	}
	context.Put(workflowKey, workflow)

	var params = common.NewMap()
	state.Put("params", params)
	if len(request.Params) > 0 {
		for k, v := range request.Params {
			if toolbox.IsString(v) {
				params[k] = context.Expand(toolbox.AsString(v))
			} else {
				params[k] = v
			}
		}
	}

	var workflowData = common.Map(workflow.Data)
	state.Put("workflow", workflowData)
	workflow.Variables.Apply(state, state, "in") // -> state to state

	for _, task := range workflow.Tasks {
		var taskAllowed, allowedServiceActions = isTaskAllowed(task, request)
		if !taskAllowed {
			continue
		}

		var hasAllowedActions = len(allowedServiceActions) > 0
		var taskActivity = &WorkflowTaskActivity{
			Task:              task.Name,
			ServiceActivities: make([]*WorkflowServiceActivity, 0),
			Data:              make(map[string]interface{}),
		}
		response.TasksActivities = append(response.TasksActivities, taskActivity)
		var taskState = common.Map(taskActivity.Data)
		state.Put("task", taskState)
		task.Variables.Apply(state, state, "in") // -> state to task state

		canRun, err := s.evaluateRunCriteria(context, task.RunCriteria)
		if err != nil {
			return nil, err
		}
		if !canRun {
			taskActivity.Skipped = fmt.Sprintf("Does not match run criteria: %v", context.Expand(task.RunCriteria))
			continue
		}

		for i, action := range task.Actions {
			if hasAllowedActions && !allowedServiceActions[i] {
				continue
			}
			serviceActivity := &WorkflowServiceActivity{
				Action:  action.Action,
				Service: action.Service,
			}
			taskActivity.ServiceActivities = append(taskActivity.ServiceActivities, serviceActivity)
			canRun, err := s.evaluateRunCriteria(context, action.RunCriteria)
			if err != nil {
				return nil, err
			}
			if !canRun {
				serviceActivity.Skipped = fmt.Sprintf("Does not match run criteria: %v", context.Expand(action.RunCriteria))
				continue
			}

			state.Put("service", action.Service)
			state.Put("action", action.Action)
			action.Variables.Apply(state, state, "in") // task state to state
			service, err := context.Service(action.Service)

			if err != nil {
				return nil, err
			}

			requestMap, err := ExpandAsMap(action.Request, state)
			if err != nil {
				return nil, err
			}
			serviceRequest, err := service.NewRequest(action.Action)
			if err != nil {
				return nil, err
			}
			serviceActivity.ServiceResponse = serviceRequest
			err = converter.AssignConverted(serviceRequest, requestMap)
			if err != nil {
				return response, err
			}

			serviceResponse := service.Run(context, serviceRequest)
			serviceActivity.ServiceResponse = serviceResponse
			if serviceResponse.Error != "" {
				if action.IgnoreError {
					serviceActivity.Error = serviceResponse.Error
				} else {
					return nil, errors.New(serviceResponse.Error)
				}
			}
			var responseMap = make(map[string]interface{})
			if serviceResponse.Response != nil {
				converter.AssignConverted(responseMap, serviceResponse.Response)
			}
			action.Variables.Apply(common.Map(responseMap), state, "out") //result to task  state
			if action.SleepInMs > 0 {
				time.Sleep(time.Millisecond * time.Duration(action.SleepInMs))
			}
		}
		task.Variables.Apply(state, state, "out") //task state to result state
	}
	workflow.Variables.Apply(state, state, "out") //task state to result state
	return response, nil
}

func (s *WorkflowService) Run(context *Context, request interface{}) *ServiceResponse {
	var response = &ServiceResponse{Status: "ok"}
	var err error
	switch actualRequest := request.(type) {
	case *WorkflowRunRequest:
		response.Response, err = s.runWorkflow(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to run workflow: %v, %v", actualRequest.Name, err)
		}
	case *WorkflowRegisterRequest:
		err := s.Register(actualRequest.Workflow)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to register workflow: %v, %v", actualRequest.Workflow.Name, err)
		}
	case *WorkflowLoadRequest:
		workflow, err := s.dao.Load(context, actualRequest.Source)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to load workflow: %v, %v", actualRequest.Source, err)
		} else {
			err = s.Register(workflow)
			if err != nil {
				response.Error = fmt.Sprintf("Failed to register workflow: %v, %v", actualRequest.Source, err)
			}
		}
	default:
		response.Error = fmt.Sprintf("Unsupported request type: %T", request)
	}
	if response.Error != "" {
		response.Status = "err"
	}
	return response
}

func (s *WorkflowService) NewRequest(action string) (interface{}, error) {
	switch action {

	case "run":
		return &WorkflowRunRequest{}, nil
	case "register":
		return &WorkflowRegisterRequest{}, nil

	case "load":
		return &WorkflowLoadRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

func NewWorkflowService() Service {
	var result = &WorkflowService{
		AbstractService: NewAbstractService(WorkflowServiceId),
		dao:             NewWorkflowDao(),
		registry:        make(map[string]*Workflow),
	}
	result.AbstractService.Service = result
	return result
}

//7.0.81
