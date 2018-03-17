package workflow

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/neatly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/storage"
	"github.com/viant/toolbox/url"
	"path"
	"strings"
	"sync"
	"time"
)

const (
	//ServiceID represents workflow Service id
	ServiceID = "workflow"

	//TaskEvalCriteriaEventType event ID
	TaskEvalCriteriaEventType = "EvalTaskCriteria"

	//ActionEvalCriteriaEventType event ID
	ActionEvalCriteriaEventType = "EvalActionCriteria"

	caller = "Workflow"
)

//Service represents a workflow service.
type Service struct {
	*endly.AbstractService
	Dao       *Dao
	registry  map[string]*endly.Workflow
	converter *toolbox.Converter
}

func (s *Service) registerWorkflow(request *RegisterRequest) (*RegisterResponse, error) {
	if err := s.Register(request.Workflow); err != nil {
		return nil, err
	}
	var response = &RegisterResponse{
		Source: request.Workflow.Source,
	}
	return response, nil
}

//Register register workflow.
func (s *Service) Register(workflow *endly.Workflow) error {
	err := workflow.Validate()
	if err != nil {
		return err
	}
	s.registry[workflow.Name] = workflow
	return nil
}

//HasWorkflow returns true if service has registered workflow.
func (s *Service) HasWorkflow(name string) bool {
	_, found := s.registry[name]
	return found
}

//Workflow returns a workflow for supplied name.
func (s *Service) Workflow(name string) (*endly.Workflow, error) {
	s.Lock()
	defer s.Unlock()
	if result, found := s.registry[name]; found {
		return result, nil
	}
	return nil, fmt.Errorf("failed to lookup workflow: %v", name)
}

func (s *Service) addVariableEvent(name string, variables endly.Variables, context *endly.Context, in, out data.Map) {
	if len(variables) == 0 {
		return
	}
	context.Publish(endly.NewModifiedStateEvent(variables, in, out))
}

func (s *Service) worflowDedicatedFolderURL(URL string) string {
	selector := WorkflowSelector(URL)
	workflowName := selector.Name()
	workflowFilename := fmt.Sprintf("%v.csv", workflowName)
	return strings.Replace(URL, workflowFilename, fmt.Sprintf("%v/%v", workflowName, workflowFilename), 1)
}

func (s *Service) getWorkflowURLs(URL string) []string {
	return []string{
		URL,
		s.worflowDedicatedFolderURL(URL),
	}
}

//getWorkflowResource returns workflow resource
func (s *Service) getWorkflowResource(state data.Map, URL string) *url.Resource {

	for _, candidate := range s.getWorkflowURLs(URL) {
		resource := url.NewResource(candidate)
		storageService, err := storage.NewServiceForURL(resource.URL, "")
		if err != nil {
			return nil
		}

		if exists, _ := storageService.Exists(candidate); exists {
			return resource
		}
	}

	if strings.Contains(URL, ":/") || strings.HasPrefix(URL, "/") {

		return nil
	}

	//Lookup shared worflow
	for _, candidate := range s.getWorkflowURLs(URL) {
		resource, err := s.Dao.NewRepoResource(state, fmt.Sprintf("workflow/%v", candidate))
		if err != nil {
			continue
		}
		storageService, err := storage.NewServiceForURL(resource.URL, "")
		if err != nil {
			return nil
		}
		if exists, _ := storageService.Exists(resource.URL); exists {
			return resource
		}
	}
	return nil
}

func (s *Service) loadWorkflowIfNeeded(context *endly.Context, name string, URL string) (err error) {
	if !s.HasWorkflow(name) {
		var workflowResource *url.Resource
		if URL == "" {
			workflowResource, err = s.Dao.NewRepoResource(context.State(), fmt.Sprintf("workflow/%v.csv", name))
			if err != nil {
				return err
			}
		} else {
			workflowResource = url.NewResource(URL)
		}

		_, err := s.loadWorkflow(context, &LoadRequest{Source: workflowResource})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) getServiceRequest(context *endly.Context, activity *endly.Activity) (endly.Service, interface{}, error) {
	var service, err = context.Service(activity.Service)
	if err != nil {
		return nil, nil, err
	}
	var state = context.State()
	activity.Request = state.Expand(activity.Request)
	request := activity.Request
	if request == nil || !toolbox.IsMap(request) {
		if toolbox.IsStruct(request) {
			var requestMap = make(map[string]interface{})

			if err = toolbox.DefaultConverter.AssignConverted(&requestMap, request); err != nil {
				return nil, nil, err
			}
			request = requestMap
		} else {
			err = fmt.Errorf("invalid request: %v, expected map but had: %T", request, request)
			return nil, nil, err
		}
	}
	requestMap := toolbox.AsMap(request)
	var serviceRequest interface{}
	serviceRequest, err = context.AsRequest(service.ID(), activity.Action, requestMap)
	if err != nil {
		return nil, nil, err
	}
	activity.Request = serviceRequest
	activity.Request = serviceRequest
	return service, serviceRequest, nil
}

func (s *Service) runAction(context *endly.Context, action *endly.ServiceAction, workflow *endly.WorkflowRun) (response interface{}, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("%v: %v", action.TagID, err)
		}
	}()
	if !workflow.CanRun() {
		return nil, nil
	}
	if err = action.ActionRequest.Validate(); err != nil {
		return nil, err
	}
	var state = context.State()

	activity := endly.NewActivity(context, action, state)
	workflow.Activity = activity
	state.Put(endly.ActivityKey, activity)

	startEvent := s.Begin(context, activity)
	defer s.End(context)(startEvent, endly.NewActivityEndEvent(activity))

	var canRun bool
	canRun, err = endly.Evaluate(context, context.State(), action.When, ActionEvalCriteriaEventType, true)
	if err != nil {
		return nil, err
	}
	if !canRun {
		activity.Ineligible = true
		return nil, nil
	}

	err = action.Init.Apply(state, state)
	s.addVariableEvent("Action.Init", action.Init, context, state, state)
	if err != nil {
		return nil, err
	}

	service, serviceRequest, err := s.getServiceRequest(context, activity)
	if err != nil {
		return nil, err
	}
	activity.ServiceResponse = service.Run(context, serviceRequest)

	if activity.ServiceResponse.Err != nil {
		return nil, activity.ServiceResponse.Err
	}
	response = activity.ServiceResponse.Response
	if response != nil && (toolbox.IsMap(response) || toolbox.IsStruct(response)) {
		s.converter.AssignConverted(&activity.Response, activity.ServiceResponse.Response)
	} else {
		activity.Response["value"] = response
	}
	var responseState = data.Map(activity.Response)
	err = action.Post.Apply(responseState, state) //result to task  state
	s.addVariableEvent("Action.Post", action.Post, context, responseState, state)
	return response, err
}

func (s *Service) injectTagIDsIfNeeded(action *endly.ActionRequest, tagIDs map[string]bool) {
	if action.Service != "workflow" || action.Action != "run" {
		return
	}
	requestMap := toolbox.AsMap(action.Request)
	requestMap["TagIDs"] = strings.Join(toolbox.MapKeysToStringSlice(tagIDs), ",")
}

func (s *Service) runTask(context *endly.Context, workflow *endly.WorkflowRun, tagIDs map[string]bool, task *endly.WorkflowTask) (data.Map, error) {
	if !workflow.CanRun() {
		return nil, nil
	}
	workflow.TaskName = task.Name
	var startTime = time.Now()
	var state = context.State()
	state.Put(endly.TaskKey, task.Name)
	canRun, err := endly.Evaluate(context, context.State(), task.When, TaskEvalCriteriaEventType, true)
	if err != nil {
		return nil, err
	}
	if !canRun {
		return nil, nil
	}
	err = task.Init.Apply(state, state)
	s.addVariableEvent("Task.Init", task.Init, context, state, state)
	if err != nil {
		return nil, err
	}
	var hasTagIDs = len(tagIDs) > 0
	var filterTagIDs = false
	if hasTagIDs {
		filterTagIDs = task.HasTagID(tagIDs)
	}

	var asyncActions = make([]*endly.ServiceAction, 0)
	for i := 0; i < len(task.Actions); i++ {
		action := task.Actions[i]
		if hasTagIDs {
			s.injectTagIDsIfNeeded(action.ActionRequest, tagIDs)
		}

		if filterTagIDs && !tagIDs[action.TagID] {
			continue
		}

		if action.Async {
			asyncActions = append(asyncActions, task.Actions[i])
			continue
		}

		var handler = func(action *endly.ServiceAction) func() (interface{}, error) {
			return func() (interface{}, error) {
				return s.runAction(context, action, workflow)
			}
		}

		moveToNextTag, err := endly.Evaluate(context, context.State(), action.Skip, "TagIdSkipCriteria", false)
		if err != nil {
			return nil, err
		}
		if moveToNextTag {
			for j := i + 1; j < len(task.Actions) && action.TagID == task.Actions[j].TagID; j++ {
				i++
			}
			continue
		}

		var extractable = make(map[string]interface{})
		repeatable := action.Repeater.Get()
		err = repeatable.Run(s.AbstractService, caller, context, handler(task.Actions[i]), extractable)
		if err != nil {
			return nil, err
		}
	}

	err = s.runAsyncActions(context, workflow, task, asyncActions)
	if err != nil {
		return nil, err
	}
	var taskPostState = data.NewMap()
	err = task.Post.Apply(state, taskPostState)
	s.addVariableEvent("Task.Post", task.Post, context, taskPostState, state)
	state.Apply(taskPostState)
	s.applyRemainingTaskSpentIfNeeded(context, task, startTime)
	return taskPostState, err
}

func (s *Service) applyRemainingTaskSpentIfNeeded(context *endly.Context, task *endly.WorkflowTask, startTime time.Time) {
	if task.TimeSpentMs > 0 {
		var elapsed = (time.Now().UnixNano() - startTime.UnixNano()) / int64(time.Millisecond)
		var remainingExecutionTime = time.Duration(task.TimeSpentMs - int(elapsed))
		s.Sleep(context, int(remainingExecutionTime))
	}
}

func (s *Service) runAsyncAction(parent, context *endly.Context, workflow *endly.WorkflowRun, action *endly.ServiceAction, group *sync.WaitGroup) error {
	defer group.Done()
	events := context.MakeAsyncSafe()
	defer events.Drain(parent)
	if _, err := s.runAction(context, action, workflow); err != nil {
		return err
	}
	var state = parent.State()
	if len(action.Post) > 0 {
		s.Lock()
		defer s.Unlock()
		var actionState = context.State()
		for _, variable := range action.Post {
			var variableName = context.Expand(variable.Name)
			state.Put(variableName, actionState.Get(variableName))
		}
	}
	return nil
}

func (s *Service) runAsyncActions(context *endly.Context, workflow *endly.WorkflowRun, task *endly.WorkflowTask, asyncAction []*endly.ServiceAction) error {
	if len(asyncAction) > 0 {
		group := &sync.WaitGroup{}
		group.Add(len(asyncAction))
		var groupErr error
		for i := range asyncAction {
			context.Publish(NewWorkflowAsyncEvent(asyncAction[i]))
			go func(action *endly.ServiceAction, actionContext *endly.Context) {
				if err := s.runAsyncAction(context, actionContext, workflow, action, group); err != nil {
					groupErr = err
				}
			}(asyncAction[i], context)
		}
		group.Wait()
		if groupErr != nil {
			return groupErr
		}
	}
	return nil
}

func (s *Service) traversePipelines(pipelines []*Pipeline, context *endly.Context, response *PipelineResponse) error {
	var state = context.State()
	for _, pipeline := range pipelines {
		if pipeline.Skip {
			continue
		}
		if pipeline.Workflow != "" {
			runRequest := NewRunRequest(pipeline.Workflow, pipeline.Params, true)
			runResponse := &RunResponse{}
			if err := endly.Run(context, runRequest, runResponse); err != nil {
				return err
			}
			if len(runResponse.Data) > 0 {
				for k, v := range runResponse.Data {
					state.Put(k, v)
				}
			}
		} else if pipeline.Action != "" {
			serviceAction := pipeline.Action
			var response = make(map[string]interface{})
			var request, err = context.AsRequest(serviceAction.Service(), serviceAction.Action(), pipeline.Params)
			if err != nil {
				return err
			}
			if err = endly.Run(context, request, response); err != nil {
				return err
			}
			if len(response) > 0 {
				for k, v := range response {
					state.Put(k, v)
				}
			}
		} else if len(pipeline.Pipelines) > 0 {
			if err := s.traversePipelines(pipeline.Pipelines, context, response); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) pipeline(context *endly.Context, request *PipelineRequest) (*PipelineResponse, error) {
	var response = &PipelineResponse{
		Response: make(map[string]interface{}),
	}
	return response, s.traversePipelines(request.Pipelines, context, response)
}

func (s *Service) pipelineWorkflows(context *endly.Context, request *PipelineRequest) (*PipelineResponse, error) {
	s.enableLoggingIfNeeded(context, request.BaseRun)
	if request.Async {
		context.Wait.Add(1)
		go func() {
			defer context.Wait.Done()
			s.pipeline(context, request)
		}()
		return &PipelineResponse{}, nil
	}
	return s.pipeline(context, request)
}

func (s *Service) runWorkflow(context *endly.Context, request *RunRequest) (response *RunResponse, err error) {
	if request.Async {
		context.Wait.Add(1)
		go func() {
			defer context.Publish(NewWorkflowEndEvent(context.SessionID))
			defer context.Wait.Done()
			_, err = s.run(context, request)
			if err != nil {
				context.Publish(endly.NewErrorEvent(fmt.Sprintf("%v", err)))
			}
		}()
		return
	}
	defer context.Publish(NewWorkflowEndEvent(context.SessionID))
	return s.run(context, request)
}

func (s *Service) enableLoggingIfNeeded(context *endly.Context, request *BaseRun) {
	if request.EnableLogging && context.Listener == nil {
		var logDirectory = path.Join(request.LogDirectory, context.SessionID)
		eventLogger := endly.NewEventLogger(logDirectory, context.Listener)
		context.Listener = eventLogger.AsEventListener()
	}
}

func (s *Service) run(upstreamContext *endly.Context, request *RunRequest) (response *RunResponse, err error) {
	response = &RunResponse{
		Data:      make(map[string]interface{}),
		SessionID: upstreamContext.SessionID,
	}
	s.enableLoggingIfNeeded(upstreamContext, request.BaseRun)
	URL := request.WorkflowURL
	if resource := s.getWorkflowResource(upstreamContext.State(), request.WorkflowURL); resource != nil {
		URL = resource.URL
	}
	err = s.loadWorkflowIfNeeded(upstreamContext, request.Name, URL)
	if err != nil {
		return response, err
	}
	workflow, err := s.Workflow(request.Name)
	if err != nil {
		return response, err
	}
	upstreamContext.Publish(NewWorkflowLoadedEvent(workflow))
	control := upstreamContext.Workflows.Push(workflow)
	defer upstreamContext.Workflows.Pop()
	parentWorkflow := upstreamContext.Workflow()
	if parentWorkflow != nil {
		upstreamContext.Put(endly.WorkflowKey, parentWorkflow)
	} else {
		upstreamContext.Put(endly.WorkflowKey, workflow)
	}

	context := upstreamContext.Clone()
	var state = context.State()

	var workflowData = data.Map(workflow.Data)
	state.Put(neatly.OwnerURL, workflow.Source.URL)
	state.Put("data", workflowData)

	params := buildParamsMap(request, context)
	if request.PublishParameters {
		for key, value := range params {
			state.Put(key, value)
		}
	}
	state.Put("params", params)

	err = workflow.Init.Apply(state, state)
	s.addVariableEvent("Workflow.Init", workflow.Init, context, state, state)
	if err != nil {
		return response, err
	}
	context.Publish(NewWorkflowInitEvent(request.Tasks, state))
	filteredTasks, err := workflow.FilterTasks(request.Tasks)
	if err != nil {
		return response, err
	}

	var tagIDs = make(map[string]bool)
	for _, tagID := range strings.Split(request.TagIDs, ",") {
		tagIDs[tagID] = true
	}

	defer s.runWorkflowDeferTaskIfNeeded(context, control)
	err = s.runWorkflowTasks(context, control, tagIDs, filteredTasks...)
	err = s.runOnErrorTaskIfNeeded(context, control, err)
	if err != nil {
		return response, err
	}

	workflow.Post.Apply(state, response.Data) //context -> workflow output
	s.addVariableEvent("Workflow.Post", workflow.Post, context, state, state)

	if workflow.SleepTimeMs > 0 {
		s.Sleep(context, workflow.SleepTimeMs)
	}
	return response, err
}

func (s *Service) runWorkflowDeferTaskIfNeeded(context *endly.Context, workflow *endly.WorkflowRun) {
	if workflow.DeferTask == "" {
		return
	}
	task, _ := workflow.Task(workflow.DeferTask)
	_ = s.runWorkflowTasks(context, workflow, nil, task)
}

func (s *Service) runOnErrorTaskIfNeeded(context *endly.Context, workflow *endly.WorkflowRun, err error) error {
	if err != nil {
		if workflow.OnErrorTask == "" {
			return err
		}
		workflow.Error = err.Error()

		var errorMap = toolbox.AsMap(workflow.WorkflowError)
		activity := workflow.WorkflowError.Activity
		if activity != nil && activity.Request != nil {
			errorMap["Request"], _ = toolbox.AsJSONText(activity.Request)
		}
		if activity != nil && activity.Response != nil {
			errorMap["Response"], _ = toolbox.AsJSONText(activity.Response)
		}
		var state = context.State()
		state.Put("error", errorMap)
		task, _ := workflow.Task(workflow.OnErrorTask)
		err = s.runWorkflowTasks(context, workflow, nil, task)
	}
	return err
}

func (s *Service) runWorkflowTasks(context *endly.Context, workflow *endly.WorkflowRun, tagIDs map[string]bool, tasks ...*endly.WorkflowTask) error {
	for _, task := range tasks {
		if workflow.IsTerminated() {
			break
		}
		if _, err := s.runTask(context, workflow, tagIDs, task); err != nil {
			return err
		}
	}
	var scheduledTask = workflow.ScheduledTask
	if scheduledTask != nil {
		workflow.ScheduledTask = nil
		return s.runWorkflowTasks(context, workflow, tagIDs, scheduledTask)
	}
	return nil
}

func buildParamsMap(request *RunRequest, context *endly.Context) data.Map {
	var params = data.NewMap()
	var state = context.State()
	if len(request.Params) > 0 {
		for k, v := range request.Params {
			params[k] = state.Expand(v)
		}
	}
	return params
}

func (s *Service) loadWorkflow(context *endly.Context, request *LoadRequest) (*LoadResponse, error) {
	workflow, err := s.Dao.Load(context, request.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to load workflow: %v, %v", request.Source.URL, err)
	}
	s.Lock()
	defer s.Unlock()
	err = s.Register(workflow)
	if err != nil {
		return nil, fmt.Errorf("failed to register workflow: %v, %v", request.Source.URL, err)
	}
	return &LoadResponse{
		Workflow: workflow,
	}, nil
}

func (s *Service) startSession(context *endly.Context) bool {
	s.RLock()
	var state = context.State()
	if state.Has(context.SessionID) {
		s.RUnlock()
		return false
	}
	s.RUnlock()
	state.Put(context.SessionID, context)
	s.Lock()
	defer s.Unlock()
	return true
}

func (s *Service) isAsyncRequest(request interface{}) bool {
	if runRequest, ok := request.(*RunRequest); ok {
		return runRequest.Async
	}
	return false
}

func (s *Service) exitWorkflow(context *endly.Context, request *ExitRequest) (*ExitResponse, error) {
	control := context.Workflows.LastRun()
	if control != nil {
		control.Terminate()
	}
	return &ExitResponse{}, nil
}

func (s *Service) runGoto(context *endly.Context, request *GotoRequest) (GotoResponse, error) {
	var response interface{}
	if len(*context.Workflows) == 0 {
		err := fmt.Errorf("no active workflow")
		return nil, err
	}
	workflow := context.Workflows.LastRun()
	var task *endly.WorkflowTask
	task, err := workflow.Task(request.Task)
	if err == nil {
		workflow.ScheduledTask = task
	}
	return response, err
}

func getServiceActivity(state data.Map) *endly.Activity {
	serviceActivity := state.Get(endly.ActivityKey)
	if serviceActivity == nil {
		return nil
	}
	if result, ok := serviceActivity.(*endly.Activity); ok {
		return result
	}
	return nil
}

func getServiceAction(state data.Map, actionRequest *endly.ActionRequest) *endly.ServiceAction {
	serviceActivity := getServiceActivity(state)
	var result = &endly.ServiceAction{
		ActionRequest: actionRequest,
		NeatlyTag:     &endly.NeatlyTag{},
	}
	if serviceActivity != nil {
		result.NeatlyTag = serviceActivity.NeatlyTag
		result.Name = serviceActivity.Action
		result.Description = serviceActivity.Description
	}
	return result
}

func getSwitchSource(context *endly.Context, sourceKey string) interface{} {
	sourceKey = context.Expand(sourceKey)
	var state = context.State()
	var result = state.Get(sourceKey)
	if result == nil {
		return nil
	}
	return toolbox.DereferenceValue(result)
}

func (s *Service) runSwitch(context *endly.Context, request *SwitchRequest) (SwitchResponse, error) {
	workflow := context.Workflows.LastRun()
	var response interface{}
	var source = getSwitchSource(context, request.SourceKey)
	matched := request.Match(source)
	if matched != nil {
		if matched.Task != "" {
			task, err := workflow.Task(matched.Task)
			if err != nil {
				return nil, err
			}
			return s.runTask(context, workflow, nil, task)
		}
		serviceAction := getServiceAction(context.State(), matched.ActionRequest)
		return s.runAction(context, serviceAction, workflow)

	}
	return response, nil
}

const (
	workflowServiceRunExample = `{
  "Name": "ec2",
  "Params": {
    "awsCredential": "${env.HOME}/.secret/aws-west.json",
    "ec2InstanceId": "i-0139209d5358e60a4"
  },
  "Tasks": "start"
}`

	workflowServiceSwitchExample = `{
  "SourceKey": "instanceState",
  "Cases": [
    {
      "Service": "aws/ec2",
      "Action": "call",
      "Request": {
        "Credential": "${env.HOME}/.secret/aws-west.json",
        "Input": {
          "InstanceIds": [
            "i-*********"
          ]
        },
        "Method": "StartInstances"
      },
      "Value": "stopped"
    },
    {
      "Service": "workflow",
      "Action": "exit",
      "Value": "running"
    }
  ]
}
`
	workflowServiceExitExample = `{}`

	workflowServiceGotoExample = `{
		"Task": "stop"
	}`
)

func (s *Service) registerRoutes() {
	s.AbstractService.Register(&endly.ServiceActionRoute{
		Action: "run",
		RequestInfo: &endly.ActionInfo{
			Description: "run workflow",
			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "run workflow",
					Data:    workflowServiceRunExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &RunRequest{}
		},
		ResponseProvider: func() interface{} {
			return &RunResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*RunRequest); ok {
				return s.runWorkflow(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.AbstractService.Register(&endly.ServiceActionRoute{
		Action: "pipeline",
		RequestInfo: &endly.ActionInfo{
			Description: "run pipeline workflow",
			Examples:    []*endly.ExampleUseCase{},
		},
		RequestProvider: func() interface{} {
			return &PipelineRequest{}
		},
		ResponseProvider: func() interface{} {
			return &PipelineResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*PipelineRequest); ok {
				return s.pipelineWorkflows(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.AbstractService.Register(&endly.ServiceActionRoute{
		Action: "load",
		RequestInfo: &endly.ActionInfo{
			Description: "load workflow from URL",
		},
		RequestProvider: func() interface{} {
			return &LoadRequest{}
		},
		ResponseProvider: func() interface{} {
			return &LoadResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*LoadRequest); ok {
				return s.loadWorkflow(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.AbstractService.Register(&endly.ServiceActionRoute{
		Action: "register",
		RequestInfo: &endly.ActionInfo{
			Description: "register workflow",
		},
		RequestProvider: func() interface{} {
			return &RegisterRequest{}
		},
		ResponseProvider: func() interface{} {
			return &LoadResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*RegisterRequest); ok {
				return s.registerWorkflow(req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.AbstractService.Register(&endly.ServiceActionRoute{
		Action: "switch",
		RequestInfo: &endly.ActionInfo{
			Description: "select action or task for matched case value",
			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "switch case",
					Data:    workflowServiceSwitchExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &SwitchRequest{}
		},
		ResponseProvider: func() interface{} {
			return struct{}{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*SwitchRequest); ok {
				return s.runSwitch(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.AbstractService.Register(&endly.ServiceActionRoute{
		Action: "goto",
		RequestInfo: &endly.ActionInfo{
			Description: "goto task",
			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "goto",
					Data:    workflowServiceGotoExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &GotoRequest{}
		},
		ResponseProvider: func() interface{} {
			return struct{}{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*GotoRequest); ok {
				return s.runGoto(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.AbstractService.Register(&endly.ServiceActionRoute{
		Action: "exit",
		RequestInfo: &endly.ActionInfo{
			Description: "exit current workflow",
			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "exit",
					Data:    workflowServiceExitExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &ExitRequest{}
		},
		ResponseProvider: func() interface{} {
			return &ExitResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*ExitRequest); ok {
				return s.exitWorkflow(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.AbstractService.Register(&endly.ServiceActionRoute{
		Action: "fail",
		RequestInfo: &endly.ActionInfo{
			Description: "fail workflow execution",
		},
		RequestProvider: func() interface{} {
			return &FailRequest{}
		},
		ResponseProvider: func() interface{} {
			return &FailResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*FailRequest); ok {
				return nil, fmt.Errorf(req.Message)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.AbstractService.Register(&endly.ServiceActionRoute{
		Action: "nop",
		RequestInfo: &endly.ActionInfo{
			Description: "iddle operation",
		},
		RequestProvider: func() interface{} {
			return &NopRequest{}
		},
		ResponseProvider: func() interface{} {
			return struct{}{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*NopRequest); ok {
				return req, nil
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.AbstractService.Register(&endly.ServiceActionRoute{
		Action: "print",
		RequestInfo: &endly.ActionInfo{
			Description: "print log message",
		},
		RequestProvider: func() interface{} {
			return &PrintRequest{}
		},
		ResponseProvider: func() interface{} {
			return struct{}{}
		},
		Handler: func(context *endly.Context, req interface{}) (interface{}, error) {
			if request, ok := req.(*PrintRequest); ok {
				if !context.CLIEnabled {
					if request.Message != "" {
						fmt.Printf("%v\n", request.Message)
					}
					if request.Error != "" {
						fmt.Printf("%v\n", request.Error)
					}
				}
				return struct{}{}, nil
			}
			return nil, fmt.Errorf("unsupported request type: %T", req)
		},
	})

}

//New returns a new workflow Service.
func New() endly.Service {
	var result = &Service{
		AbstractService: endly.NewAbstractService(ServiceID),
		Dao:             NewDao(),
		registry:        make(map[string]*endly.Workflow),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
