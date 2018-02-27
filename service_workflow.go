package endly

import (
	"fmt"
	"github.com/viant/neatly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
	"path"
	"strings"
	"sync"
	"time"
)

const (
	//WorkflowServiceID represents workflow service id
	WorkflowServiceID = "workflow"

	//WorkflowTaskEvalCriteriaEventType event ID
	WorkflowTaskEvalCriteriaEventType = "EvalTaskCriteria"

	//WorkflowActionEvalCriteriaEventType event ID
	WorkflowActionEvalCriteriaEventType = "EvalActionCriteria"

	//WorkflowServiceActivityKey Activity key
	WorkflowServiceActivityKey = "Activity"

	workflowCaller = "Workflow"

	workflowError = "error"
)

//WorkflowServiceActivityEndEventType represents Activity end event type.
type WorkflowServiceActivityEndEventType struct {
}

type workflowService struct {
	*AbstractService
	Dao      *WorkflowDao
	registry map[string]*Workflow
}

func (s *workflowService) registerWorkflow(request *WorkflowRegisterRequest) (*WorkflowRegisterResponse, error) {
	if err := s.Register(request.Workflow); err != nil {
		return nil, err
	}
	var response = &WorkflowRegisterResponse{
		Source: request.Workflow.Source,
	}
	return response, nil
}

func (s *workflowService) Register(workflow *Workflow) error {
	err := workflow.Validate()
	if err != nil {
		return err
	}
	s.registry[workflow.Name] = workflow
	return nil
}

func (s *workflowService) HasWorkflow(name string) bool {
	_, found := s.registry[name]
	return found
}

func (s *workflowService) Workflow(name string) (*Workflow, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if result, found := s.registry[name]; found {
		return result, nil
	}
	return nil, fmt.Errorf("failed to lookup workflow: %v", name)
}

func (s *workflowService) addVariableEvent(name string, variables Variables, context *Context, in, out data.Map) {
	if len(variables) == 0 {
		return
	}
	var sources = make(map[string]interface{})
	var values = make(map[string]interface{})
	for _, variable := range variables {
		if variable.From != "" {
			from := strings.Replace(variable.From, "<-", "", 1)
			from = strings.Replace(from, "++", "", 1)
			sources[from], _ = in.GetValue(from)
		}
		var name = variable.Name
		name = strings.Replace(name, "->", "", 1)
		name = strings.Replace(name, "++", "", 1)
		values[name], _ = in.GetValue(name)

	}
	AddEvent(context, name, Pairs("variables", variables, "values", values, "sources", sources), Debug)
}

func (s *workflowService) loadWorkflowIfNeeded(context *Context, name string, URL string) (err error) {
	if !s.HasWorkflow(name) {
		var workflowResource *url.Resource
		if URL == "" {
			workflowResource, err = s.Dao.NewRepoResource(context.state, fmt.Sprintf("workflow/%v.csv", name))
			if err != nil {
				return err
			}
		} else {
			workflowResource = url.NewResource(URL)
		}

		_, err := s.loadWorkflow(context, &WorkflowLoadRequest{Source: workflowResource})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *workflowService) getServiceRequest(context *Context, activity *WorkflowServiceActivity) (Service, interface{}, error) {
	var service, err = context.Service(activity.Service)
	if err != nil {
		return nil, nil, err
	}
	var state = context.state
	activity.Request = state.Expand(activity.Request)
	request := activity.Request
	if request == nil || !toolbox.IsMap(request) {
		if toolbox.IsStruct(request) {
			var requestMap = make(map[string]interface{})
			converter := toolbox.NewColumnConverter("")
			if err = converter.AssignConverted(&requestMap, request); err != nil {
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

func (s *workflowService) runAction(context *Context, action *ServiceAction, workflow *WorkflowControl) (response interface{}, err error) {
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
	var state = context.state

	serviceActivity := NewWorkflowServiceActivity(context, action, state)
	workflow.Activity = serviceActivity
	state.Put(WorkflowServiceActivityKey, serviceActivity)

	startEvent := s.Begin(context, action, Pairs(WorkflowServiceActivityKey, serviceActivity), Info)
	defer s.End(context)(startEvent, Pairs("value", &WorkflowServiceActivityEndEventType{}, "response", serviceActivity.Response))
	var canRun bool
	canRun, err = EvaluateCriteria(context, action.RunCriteria, WorkflowActionEvalCriteriaEventType, true)
	if err != nil {
		return nil, err
	}
	if !canRun {
		serviceActivity.Ineligible = true
		return nil, nil
	}

	err = action.Init.Apply(state, state)
	s.addVariableEvent("Action.Init", action.Init, context, state, state)
	if err != nil {
		return nil, err
	}

	service, serviceRequest, err := s.getServiceRequest(context, serviceActivity)
	if err != nil {
		return nil, err
	}
	serviceActivity.ServiceResponse = service.Run(context, serviceRequest)

	if serviceActivity.ServiceResponse.err != nil {
		return nil, serviceActivity.ServiceResponse.err
	}
	response = serviceActivity.ServiceResponse.Response
	if response != nil && (toolbox.IsMap(response) || toolbox.IsStruct(response)) {
		converter.AssignConverted(&serviceActivity.Response, serviceActivity.ServiceResponse.Response)
	} else {
		serviceActivity.Response["value"] = response
	}
	var responseState = data.Map(serviceActivity.Response)
	err = action.Post.Apply(responseState, state) //result to task  state
	s.addVariableEvent("Action.Post", action.Post, context, responseState, state)
	return response, err
}

func (s *workflowService) injectTagIdsIdNeeded(action *ActionRequest, tagIDs map[string]bool) {
	if action.Service != "workflow" || action.Action != "run" {
		return
	}
	requestMap := toolbox.AsMap(action.Request)
	requestMap["TagIDs"] = strings.Join(toolbox.MapKeysToStringSlice(tagIDs), ",")
}

func (s *workflowService) runTask(context *Context, workflow *WorkflowControl, tagIDs map[string]bool, task *WorkflowTask) (data.Map, error) {
	if !workflow.CanRun() {
		return nil, nil
	}
	workflow.TaskName = task.Name
	var startTime = time.Now()
	var state = context.state
	state.Put(":task", task)
	canRun, err := EvaluateCriteria(context, task.RunCriteria, WorkflowTaskEvalCriteriaEventType, true)
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
	startEvent := s.Begin(context, task, Pairs("ID", task.Name, "state", state.AsEncodableMap()))
	defer s.End(context)(startEvent, Pairs())

	var hasTagIDs = len(tagIDs) > 0
	var filterTagIDs = false
	if hasTagIDs {
		filterTagIDs = task.HasTagID(tagIDs)
	}

	var asyncActions = make([]*ServiceAction, 0)
	for i := 0; i < len(task.Actions); i++ {
		action := task.Actions[i]
		if hasTagIDs {
			s.injectTagIdsIdNeeded(action.ActionRequest, tagIDs)
		}

		if filterTagIDs && !tagIDs[action.TagID] {
			continue
		}

		if action.Async {
			asyncActions = append(asyncActions, task.Actions[i])
			var asyncEvent = NewAsyncServiceActionEvent(workflow.Name, context.Expand(task.Name), context.Expand(action.Description), action.Service, action.Action, action.TagID)
			AddEvent(context, asyncEvent, Pairs("value", asyncEvent))
			continue
		}

		var handler = func(action *ServiceAction) func() (interface{}, error) {
			return func() (interface{}, error) {
				return s.runAction(context, action, workflow)
			}
		}

		moveToNextTag, err := EvaluateCriteria(context, action.SkipCriteria, "TagIdSkipCriteria", false)
		if err != nil {
			return nil, err
		}
		if moveToNextTag {
			for j := i + 1; j < len(task.Actions) && action.TagID == task.Actions[j].TagID; j++ {
				i++
			}
			continue
		}

		var extractable = make(map[string]string)
		repeatable := action.Repeatable.Get()
		err = repeatable.Run(s.AbstractService, workflowCaller, context, handler(task.Actions[i]), extractable)
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

func (s *workflowService) applyRemainingTaskSpentIfNeeded(context *Context, task *WorkflowTask, startTime time.Time) {
	if task.TimeSpentMs > 0 {
		var elapsed = (time.Now().UnixNano() - startTime.UnixNano()) / int64(time.Millisecond)
		var remainingExecutionTime = time.Duration(task.TimeSpentMs - int(elapsed))
		s.Sleep(context, int(remainingExecutionTime))
	}
}

func (s *workflowService) runAsyncActions(context *Context, workflow *WorkflowControl, task *WorkflowTask, asyncAction []*ServiceAction) error {
	var err error
	var state = context.state
	if len(asyncAction) > 0 {
		group := sync.WaitGroup{}
		group.Add(len(asyncAction))

		var groupErr error
		for _, action := range asyncAction {
			go func(actionContext *Context, action *ServiceAction) {
				defer group.Done()
				actionContext.MakeAsyncSafe()
				_, err = s.runAction(actionContext, action, workflow)
				if err != nil {
					groupErr = fmt.Errorf("failed to run action:%v %v", action.Tag, err)
				}
				if len(action.Post) > 0 {
					var actionState = actionContext.state
					for _, variable := range action.Post {
						var variableName = context.Expand(variable.Name)
						state.Put(variableName, actionState.Get(variableName))
					}
				}
				s.publishEvents(context, actionContext.Events.Events)
			}(context.Clone(), action)
		}

		group.Wait()

		if groupErr != nil {
			return groupErr
		}
	}
	return err
}
func (s *workflowService) publishEvents(context *Context, events []*Event) {
	if len(events) > 0 {
		s.mutex.Lock()
		defer s.mutex.Unlock()
		for _, event := range events {
			context.Events.Push(event)
			if context.EventLogger != nil {
				context.EventLogger.Log(event)
			}
		}
	}
}

func (s *workflowService) runWorkflow(context *Context, request *WorkflowRunRequest) (response *WorkflowRunResponse, err error) {
	startedSession := s.startSession(context)
	if request.Async {
		go func() {
			_, err = s.run(context, request)
			if err != nil {
				var eventType = &ErrorEventType{Error: fmt.Sprintf("%v", err)}
				AddEvent(context, eventType, Pairs("value", eventType), Info)
			}
			if startedSession {
				defer s.removeSession(context)
			}
		}()
	} else {
		return s.run(context, request)
	}
	return &WorkflowRunResponse{
		Data:      make(map[string]interface{}),
		SessionID: context.SessionID,
	}, nil
}

func (s *workflowService) run(upstreamContext *Context, request *WorkflowRunRequest) (response *WorkflowRunResponse, err error) {
	response = &WorkflowRunResponse{
		Data:      make(map[string]interface{}),
		SessionID: upstreamContext.SessionID,
	}
	if request.EnableLogging {
		upstreamContext.EventLogger = NewEventLogger(path.Join(request.LoggingDirectory, upstreamContext.SessionID))
	}
	err = s.loadWorkflowIfNeeded(upstreamContext, request.Name, request.WorkflowURL)
	if err != nil {
		return response, err
	}
	workflow, err := s.Workflow(request.Name)
	if err != nil {
		return response, err
	}
	AddEvent(upstreamContext, "Workflow.Loaded", Pairs("workflow", workflow))
	control := upstreamContext.Workflows.Push(workflow)
	defer upstreamContext.Workflows.Pop()
	parentWorkflow := upstreamContext.Workflow()
	if parentWorkflow != nil {
		upstreamContext.Put(workflowKey, parentWorkflow)
	} else {
		upstreamContext.Put(workflowKey, workflow)
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
	AddEvent(context, "State.Init", Pairs("state", state.AsEncodableMap(), "tasks", request.Tasks), Debug)
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

func (s *workflowService) runWorkflowDeferTaskIfNeeded(context *Context, workflow *WorkflowControl) {
	if workflow.DeferTask == "" {
		return
	}
	task, _ := workflow.Task(workflow.DeferTask)
	_ = s.runWorkflowTasks(context, workflow, nil, task)
}

func (s *workflowService) runOnErrorTaskIfNeeded(context *Context, workflow *WorkflowControl, err error) error {
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
		context.state.Put(workflowError, errorMap)
		task, _ := workflow.Task(workflow.OnErrorTask)
		err = s.runWorkflowTasks(context, workflow, nil, task)
	}
	return err
}

func (s *workflowService) runWorkflowTasks(context *Context, workflow *WorkflowControl, tagIDs map[string]bool, tasks ...*WorkflowTask) error {
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

func buildParamsMap(request *WorkflowRunRequest, context *Context) data.Map {
	var params = data.NewMap()
	var state = context.state
	if len(request.Params) > 0 {
		for k, v := range request.Params {
			params[k] = state.Expand(v)
		}
	}
	return params
}

func (s *workflowService) loadWorkflow(context *Context, request *WorkflowLoadRequest) (*WorkflowLoadResponse, error) {
	workflow, err := s.Dao.Load(context, request.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to load workflow: %v, %v", request.Source.URL, err)
	}
	s.Mutex().Lock()
	defer s.Mutex().Unlock()
	err = s.Register(workflow)
	if err != nil {
		return nil, fmt.Errorf("failed to register workflow: %v, %v", request.Source.URL, err)
	}
	return &WorkflowLoadResponse{
		Workflow: workflow,
	}, nil
}

func (s *workflowService) removeSession(context *Context) {
	go func() {
		time.Sleep(2 * time.Second)
		s.Mutex().Lock()
		defer s.Mutex().Unlock()
		s.state.Delete(context.SessionID)
	}()
}

func (s *workflowService) startSession(context *Context) bool {
	s.Mutex().RLock()
	if s.state.Has(context.SessionID) {
		s.Mutex().RUnlock()
		return false
	}
	s.Mutex().RUnlock()
	s.state.Put(context.SessionID, context)
	s.Mutex().Lock()
	defer s.Mutex().Unlock()
	return true
}

func (s *workflowService) isAsyncRequest(request interface{}) bool {
	if runRequest, ok := request.(*WorkflowRunRequest); ok {
		return runRequest.Async
	}
	return false
}

func (s *workflowService) exitWorkflow(context *Context, request *WorkflowExitRequest) (*WorkflowExitResponse, error) {
	control := context.Workflows.LastControl()
	if control != nil {
		control.Terminate()
	}
	return &WorkflowExitResponse{}, nil
}

func (s *workflowService) runGoto(context *Context, request *WorkflowGotoRequest) (WorkflowGotoResponse, error) {
	var response interface{}
	if len(*context.Workflows) == 0 {
		err := fmt.Errorf("no active workflow")
		return nil, err
	}
	workflow := context.Workflows.LastControl()
	var task *WorkflowTask
	task, err := workflow.Task(request.Task)
	if err == nil {
		workflow.ScheduledTask = task
	}
	return response, err
}

func getServiceActivity(state data.Map) *WorkflowServiceActivity {
	serviceActivity := state.Get(WorkflowServiceActivityKey)
	if serviceActivity == nil {
		return nil
	}
	if result, ok := serviceActivity.(*WorkflowServiceActivity); ok {
		return result
	}
	return nil
}

func getServiceAction(state data.Map, actionRequest *ActionRequest) *ServiceAction {
	serviceActivity := getServiceActivity(state)
	var result = &ServiceAction{
		ActionRequest: actionRequest,
		NeatlyTag:     &NeatlyTag{},
	}
	if serviceActivity != nil {
		result.NeatlyTag = serviceActivity.NeatlyTag
		result.Name = serviceActivity.Action
		result.Description = serviceActivity.Description
	}
	return result
}

func getSwitchSource(context *Context, sourceKey string) interface{} {
	sourceKey = context.Expand(sourceKey)
	var state = context.state
	var result = state.Get(sourceKey)
	if result == nil {
		return nil
	}
	return toolbox.DereferenceValue(result)
}

func (s *workflowService) runSwitch(context *Context, request *WorkflowSwitchRequest) (WorkflowSwitchResponse, error) {
	workflow := context.Workflows.LastControl()
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
		serviceAction := getServiceAction(context.state, matched.ActionRequest)
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

func (s *workflowService) registerRoutes() {
	s.AbstractService.Register(&ServiceActionRoute{
		Action: "run",
		RequestInfo: &ActionInfo{
			Description: "run workflow",
			Examples: []*ExampleUseCase{
				{
					UseCase: "run workflow",
					Data:    workflowServiceRunExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &WorkflowRunRequest{}
		},
		ResponseProvider: func() interface{} {
			return &WorkflowRunResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*WorkflowRunRequest); ok {
				return s.runWorkflow(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.AbstractService.Register(&ServiceActionRoute{
		Action: "load",
		RequestInfo: &ActionInfo{
			Description: "load workflow from URL",
		},
		RequestProvider: func() interface{} {
			return &WorkflowLoadRequest{}
		},
		ResponseProvider: func() interface{} {
			return &WorkflowLoadResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*WorkflowLoadRequest); ok {
				return s.loadWorkflow(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.AbstractService.Register(&ServiceActionRoute{
		Action: "register",
		RequestInfo: &ActionInfo{
			Description: "register workflow",
		},
		RequestProvider: func() interface{} {
			return &WorkflowRegisterRequest{}
		},
		ResponseProvider: func() interface{} {
			return &WorkflowLoadResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*WorkflowRegisterRequest); ok {
				return s.registerWorkflow(handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.AbstractService.Register(&ServiceActionRoute{
		Action: "switch",
		RequestInfo: &ActionInfo{
			Description: "select action or task for matched case value",
			Examples: []*ExampleUseCase{
				{
					UseCase: "switch case",
					Data:    workflowServiceSwitchExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &WorkflowSwitchRequest{}
		},
		ResponseProvider: func() interface{} {
			return struct{}{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*WorkflowSwitchRequest); ok {
				return s.runSwitch(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.AbstractService.Register(&ServiceActionRoute{
		Action: "goto",
		RequestInfo: &ActionInfo{
			Description: "goto task",
			Examples: []*ExampleUseCase{
				{
					UseCase: "goto",
					Data:    workflowServiceGotoExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &WorkflowGotoRequest{}
		},
		ResponseProvider: func() interface{} {
			return struct{}{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*WorkflowGotoRequest); ok {
				return s.runGoto(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.AbstractService.Register(&ServiceActionRoute{
		Action: "exit",
		RequestInfo: &ActionInfo{
			Description: "exit current workflow",
			Examples: []*ExampleUseCase{
				{
					UseCase: "exit",
					Data:    workflowServiceExitExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &WorkflowExitRequest{}
		},
		ResponseProvider: func() interface{} {
			return &WorkflowExitResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*WorkflowExitRequest); ok {
				return s.exitWorkflow(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.AbstractService.Register(&ServiceActionRoute{
		Action: "fail",
		RequestInfo: &ActionInfo{
			Description: "fail workflow execution",
		},
		RequestProvider: func() interface{} {
			return &WorkflowFailRequest{}
		},
		ResponseProvider: func() interface{} {
			return &WorkflowFailResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*WorkflowFailRequest); ok {
				return nil, fmt.Errorf(handlerRequest.Message)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

}

//NewWorkflowService returns a new workflow service.
func NewWorkflowService() Service {
	var result = &workflowService{
		AbstractService: NewAbstractService(WorkflowServiceID),
		Dao:             NewWorkflowDao(),
		registry:        make(map[string]*Workflow),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
