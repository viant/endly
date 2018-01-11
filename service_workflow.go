package endly

import (
	"errors"
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

	//WorkflowServiceRunAction represents workflow run action
	WorkflowServiceRunAction = "run"

	//WorkflowServiceSwitchTaskAction represents workflow switch task action
	WorkflowServiceSwitchTaskAction = "switch-task"

	//WorkflowServiceSwitcActionAction represents workflow switch action
	WorkflowServiceSwitcActionAction = "switch-action"

	//WorkflowServiceRepeatTaskAction represents workflow repeat task action
	WorkflowServiceRepeatTaskAction = "repeat-task"

	//WorkflowServiceRepeatActionAction represents workflow repeat action
	WorkflowServiceRepeatActionAction = "repeat-action"

	//WorkflowServiceRegisterAction represents workflow register action
	WorkflowServiceRegisterAction = "register"

	//WorkflowServiceLoadAction represents workflow load action
	WorkflowServiceLoadAction = "load"

	//WorkflowServiceActivityKey activity key
	WorkflowServiceActivityKey = "activity"

	workflowServiceCaller = "Workflow"
	workflowError         = "error"
)

//WorkflowServiceActivityEndEventType represents activity end event type.
type WorkflowServiceActivityEndEventType struct {
}

type workflowService struct {
	*AbstractService
	Dao      *WorkflowDao
	registry map[string]*Workflow
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

func (s *workflowService) addVariableEvent(name string, variables Variables, context *Context, state data.Map) {
	if len(variables) == 0 {
		return
	}
	var sources = make(map[string]interface{})
	var values = make(map[string]interface{})
	for _, variable := range variables {
		if variable.From != "" {
			from := strings.Replace(variable.From, "<-", "", 1)
			from = strings.Replace(from, "++", "", 1)
			sources[from], _ = state.GetValue(from)
		}
		var name = variable.Name
		name = strings.Replace(name, "->", "", 1)
		name = strings.Replace(name, "++", "", 1)
		values[name], _ = state.GetValue(name)
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

func (s *workflowService) asServiceRequest(action *ServiceAction, serviceRequest interface{}, requestMap map[string]interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("failed to cast %v into %T, due to %v", requestMap, serviceRequest, r)
		}
	}()

	err = converter.AssignConverted(serviceRequest, requestMap)
	if err != nil {
		return fmt.Errorf("failed to create request %v on %v.%v, %v", requestMap, action.Service, action.Action, err)
	}
	return err

}

func (s *workflowService) runAction(context *Context, action *ServiceAction) (*WorkflowServiceActivity, error) {
	if err := action.ActionRequest.Validate(); err != nil {
		return nil, err
	}
	var state = context.state
	serviceActivity := NewWorkflowServiceActivity(context, action)
	state.Put(WorkflowServiceActivityKey, serviceActivity)
	startEvent := s.Begin(context, action, Pairs(WorkflowServiceActivityKey, serviceActivity), Info)
	defer s.End(context)(startEvent, Pairs("value", &WorkflowServiceActivityEndEventType{}, "response", serviceActivity.Response))
	canRun, err := EvaluateCriteria(context, action.RunCriteria, WorkflowActionEvalCriteriaEventType, true)
	if err != nil {
		return nil, err
	}
	if !canRun {
		serviceActivity.Ineligible = true
		return nil, nil
	}
	err = action.Init.Apply(state, state)
	s.addVariableEvent("Action.Init", action.Init, context, state)
	if err != nil {
		return nil, err
	}
	service, err := context.Service(action.Service)
	if err != nil {
		return nil, err
	}

	expandedRequest := state.Expand(action.Request)
	if expandedRequest == nil || !toolbox.IsMap(expandedRequest) {
		return nil, fmt.Errorf("failed to evaluate request: %v, expected map but had: %T", expandedRequest, expandedRequest)
	}

	requestMap := toolbox.AsMap(expandedRequest)
	serviceRequest, err := service.NewRequest(action.Action)
	if err != nil {
		return nil, err
	}
	serviceActivity.Request = serviceRequest
	err = s.asServiceRequest(action, serviceRequest, requestMap)
	if err != nil {
		return nil, fmt.Errorf("failed to create service request: %v %v", requestMap, err)
	}
	serviceResponse := service.Run(context, serviceRequest)
	serviceActivity.ServiceResponse = serviceResponse

	if serviceResponse.Error != "" {
		var err = reportError(errors.New(serviceResponse.Error))
		return nil, err
	}

	var response = serviceResponse.Response
	if response != nil && (toolbox.IsMap(response) || toolbox.IsStruct(response)) {
		converter.AssignConverted(&serviceActivity.Response, serviceResponse.Response)
	} else {
		serviceActivity.Response["value"] = response
	}
	err = action.Post.Apply(data.Map(serviceActivity.Response), state) //result to task  state
	s.addVariableEvent("Action.Post", action.Post, context, state)
	if err != nil {
		return nil, err
	}
	s.Sleep(context, int(action.SleepTimeMs))
	return serviceActivity, nil
}

func (s *workflowService) runTask(context *Context, workflow *Workflow, task *WorkflowTask) (data.Map, error) {
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
	s.addVariableEvent("Task.Init", task.Init, context, state)
	if err != nil {
		return nil, err
	}
	startEvent := s.Begin(context, task, Pairs("ID", task.Name))
	defer s.End(context)(startEvent, Pairs())

	var asyncActions = make([]*ServiceAction, 0)
	for i := 0; i < len(task.Actions); i++ {
		action := task.Actions[i]
		if action.Async {
			asyncActions = append(asyncActions, action)
			var asyncEvent = &AsyncServiceActionEvent{
				Workflow:    workflow.Name,
				Task:        context.Expand(task.Name),
				Description: context.Expand(action.Description),
				Service:     action.Service,
				Action:      action.Action,
				TagID:       action.TagID,
			}
			AddEvent(context, asyncEvent, Pairs("value", asyncEvent))
			continue
		}

		_, err = s.runAction(context, action)
		if err != nil {
			return nil, fmt.Errorf("failed to run action:%v %v", action.Tag, err)
		}

		moveToNextTag, err := EvaluateCriteria(context, action.SkipCriteria, "TagIdSkipCriteria", false)
		if err != nil {
			return nil, err
		}
		if moveToNextTag {
			for j := i + 1; j < len(task.Actions) && action.TagID == task.Actions[j].TagID; j++ {
				i++
			}
		}
	}
	err = s.runAsyncActions(context, workflow, task, asyncActions)
	if err != nil {
		return nil, err
	}
	var taskPostState = data.NewMap()
	err = task.Post.Apply(state, taskPostState)
	s.addVariableEvent("Task.Post", task.Post, context, taskPostState)
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

func (s *workflowService) runAsyncActions(context *Context, workflow *Workflow, task *WorkflowTask, asyncAction []*ServiceAction) error {
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
				_, err = s.runAction(actionContext, action)
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

func (s *workflowService) runWorkflow(upstreamContext *Context, request *WorkflowRunRequest) (*WorkflowRunResponse, error) {
	if request.EnableLogging {
		upstreamContext.EventLogger = NewEventLogger(path.Join(request.LoggingDirectory, upstreamContext.SessionID))
	}

	var err = s.loadWorkflowIfNeeded(upstreamContext, request.Name, request.WorkflowURL)
	if err != nil {
		return nil, err
	}

	workflow, err := s.Workflow(request.Name)
	if err != nil {
		return nil, err
	}
	AddEvent(upstreamContext, "Workflow.Loaded", Pairs("workflow", workflow))
	upstreamContext.Workflows.Push(workflow)
	defer upstreamContext.Workflows.Pop()
	var response = &WorkflowRunResponse{
		SessionID: upstreamContext.SessionID,
		Data:      make(map[string]interface{}),
	}

	parentWorkflow := upstreamContext.Workflow()
	if parentWorkflow != nil {
		upstreamContext.Put(workflowKey, parentWorkflow)
	} else {
		upstreamContext.Put(workflowKey, workflow)
	}

	context := upstreamContext.Clone()
	var state = context.State()

	if workflow.Source.URL == "" {
		return nil, fmt.Errorf("workflow.Source was empty %v", workflow.Name)
	}

	var workflowData = data.Map(workflow.Data)
	state.Put(neatly.OwnerURL, workflow.Source.URL)
	state.Put("data", workflowData)

	params := buildParamsMap(request, context)
	if request.PublishParameters {
		for key, value := range params {
			state.Put(key, state.Expand(value))
		}
	}
	state.Put("params", params)
	err = workflow.Init.Apply(state, state)
	s.addVariableEvent("Workflow.Init", workflow.Init, context, state)
	if err != nil {
		return nil, err
	}
	AddEvent(context, "State.Init", Pairs("state", state.AsEncodableMap()), Debug)
	filteredTask, err := workflow.FilterTasks(request.Tasks)
	if err != nil {
		return nil, err
	}
	for _, task := range filteredTask {
		_, err = s.runTask(context, workflow, task)
		if err != nil {
			if workflow.OnErrorTask == "" {
				return nil, err
			}
			state.Put(workflowError, err.Error())
			onErrorTask, err := workflow.Task(workflow.OnErrorTask)
			if onErrorTask != nil {
				_, err = s.runTask(context, workflow, onErrorTask)
			}
			if err != nil {
				return nil, err
			}
		}
	}
	workflow.Post.Apply(state, response.Data) //context -> workflow output
	s.addVariableEvent("Workflow.Post", workflow.Post, context, state)

	if workflow.SleepTimeMs > 0 {
		s.Sleep(context, workflow.SleepTimeMs)
	}
	return response, nil
}

func buildParamsMap(request *WorkflowRunRequest, context *Context) data.Map {
	var params = data.NewMap()
	if len(request.Params) > 0 {
		for k, v := range request.Params {
			if toolbox.IsString(v) {
				params[k] = context.Expand(toolbox.AsString(v))
			} else {
				params[k] = v
			}
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

func (s *workflowService) reportErrorIfNeeded(context *Context, response *ServiceResponse) {
	if response.Error != "" {
		var errorEventType = &ErrorEventType{Error: response.Error}
		AddEvent(context, errorEventType, Pairs("value", errorEventType), Info)
	}
}

func (s *workflowService) Run(context *Context, request interface{}) *ServiceResponse {
	startedSession := s.startSession(context)
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok"}
	defer s.reportErrorIfNeeded(context, response)

	if !s.isAsyncRequest(request) {
		defer s.End(context)(startEvent, Pairs("response", response))
	}
	var err error
	var errorTemplate = "%v"
	switch actualRequest := request.(type) {
	case *WorkflowRunRequest:
		if actualRequest.Async {
			go func() {
				if startedSession {
					defer s.reportErrorIfNeeded(context, response)
					defer s.removeSession(context)
				}
				_, err := s.runWorkflow(context, actualRequest)
				if err != nil {
					var eventType = &ErrorEventType{Error: fmt.Sprintf("%v", err)}
					AddEvent(context, eventType, Pairs("value", eventType), Info)
				}
				s.End(context)(startEvent, Pairs("response", response))
			}()
			response.Response = &WorkflowRunResponse{
				SessionID: context.SessionID,
			}
			return response
		}
		response.Response, err = s.runWorkflow(context, actualRequest)
		errorTemplate = fmt.Sprintf("failed to run workflow: %v, %v", actualRequest.Name, "%v")
	case *WorkflowRegisterRequest:
		err := s.Register(actualRequest.Workflow)
		errorTemplate = fmt.Sprintf("failed to register workflow: %v, %v", actualRequest.Workflow.Name, err)

	case *WorkflowLoadRequest:
		response.Response, err = s.loadWorkflow(context, actualRequest)
		errorTemplate = "%v"

	case *WorkflowRepeatActionRequest:
		response.Response, err = s.repeatAction(context, actualRequest)
		errorTemplate = "%v"

	case *WorkflowRepeatTaskRequest:
		response.Response, err = s.repeatTask(context, actualRequest)
		errorTemplate = "%v"

	case *WorkflowSwitchActionRequest:
		response.Response, err = s.switchAction(context, actualRequest)
		errorTemplate = "%v"

	case *WorkflowSwitchTaskRequest:
		response.Response, err = s.switchTask(context, actualRequest)
		errorTemplate = "%v"

	default:
		err = fmt.Errorf("unsupported request type: %T", request)
	}

	if err != nil {
		response.Status = "error"
		response.Error = fmt.Sprintf(errorTemplate, err)
	}

	return response
}

func (s *workflowService) NewRequest(action string) (interface{}, error) {
	switch action {
	case WorkflowServiceRunAction:
		return &WorkflowRunRequest{}, nil
	case WorkflowServiceRegisterAction:
		return &WorkflowRegisterRequest{}, nil
	case WorkflowServiceLoadAction:
		return &WorkflowLoadRequest{}, nil
	case WorkflowServiceSwitcActionAction:
		return &WorkflowSwitchActionRequest{}, nil
	case WorkflowServiceSwitchTaskAction:
		return &WorkflowSwitchTaskRequest{}, nil
	case WorkflowServiceRepeatActionAction:
		return &WorkflowRepeatActionRequest{}, nil
	case WorkflowServiceRepeatTaskAction:
		return &WorkflowRepeatTaskRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
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

func getServiceTask(workflow *Workflow, task string) (*WorkflowTask, error) {
	for _, candidate := range workflow.Tasks {
		if candidate.Name == task {
			return candidate, nil
		}
	}

	return nil, fmt.Errorf("failed to lookup task %v in %v (%v)", task, workflow.Name, workflow.Source.URL)
}

func (s *workflowService) repeatAction(context *Context, request *WorkflowRepeatActionRequest) (*WorkflowRepeatActionResponse, error) {
	var state = context.state
	var response = &WorkflowRepeatActionResponse{
		Extracted: make(map[string]string),
	}
	serviceAction := getServiceAction(state, request.ActionRequest)
	var handler = func() (interface{}, error) {
		response.Repeated++
		response, err := s.runAction(context, serviceAction)
		if err != nil {
			return nil, err
		}
		return response.ServiceResponse, nil
	}
	repetable := request.Repeatable.Get()
	err := repetable.Run(workflowServiceCaller, context, handler, response.Extracted)
	return response, err
}

func (s *workflowService) repeatTask(context *Context, request *WorkflowRepeatTaskRequest) (*WorkflowRepeatTaskResponse, error) {
	var response = &WorkflowRepeatTaskResponse{
		Extracted: make(map[string]string),
	}
	workflow := context.Workflows.Last()
	task, err := getServiceTask(workflow, request.Task)
	if err != nil {
		return nil, err
	}
	var handler = func() (interface{}, error) {
		response.Repeated++
		postState, err := s.runTask(context, workflow, task)
		if err != nil {
			return nil, err
		}
		return postState, nil
	}
	repetable := request.Repeatable.Get()
	err = repetable.Run(workflowServiceCaller, context, handler, response.Extracted)
	return response, err
}

func getSwitchSource(context *Context, sourceKey string) interface{} {
	sourceKey = context.Expand(sourceKey)
	var state = context.state
	return state.Get(sourceKey)
}

func (s *workflowService) switchAction(context *Context, request *WorkflowSwitchActionRequest) (*WorkflowSwitchActionResponse, error) {
	var response = &WorkflowSwitchActionResponse{}
	var source = getSwitchSource(context, request.SourceKey)
	actionRequest := request.Match(source)
	if actionRequest != nil {
		response.Service = actionRequest.Service
		response.Action = actionRequest.Action
		serviceAction := getServiceAction(context.state, actionRequest)
		activity, err := s.runAction(context, serviceAction)
		if err != nil {
			return nil, err
		}
		response.Response = activity.Response
	}
	return response, nil
}

func (s *workflowService) switchTask(context *Context, request *WorkflowSwitchTaskRequest) (*WorkflowSwitchTaskResponse, error) {
	var response = &WorkflowSwitchTaskResponse{}
	var source = getSwitchSource(context, request.SourceKey)
	taskName := request.Match(source)
	if taskName == "" {
		return response, nil
	}
	response.Task = taskName
	workflow := context.Workflows.Last()
	task, err := getServiceTask(workflow, taskName)
	if err != nil {
		return nil, err
	}
	_, err = s.runTask(context, workflow, task)
	if err != nil {
		return nil, err
	}
	return response, nil
}

//NewWorkflowService returns a new workflow service.
func NewWorkflowService() Service {
	var result = &workflowService{
		AbstractService: NewAbstractService(WorkflowServiceID,
			WorkflowServiceRunAction,
			WorkflowServiceRegisterAction,
			WorkflowServiceLoadAction,
			WorkflowServiceSwitcActionAction,
			WorkflowServiceSwitchTaskAction,
			WorkflowServiceRepeatTaskAction,
			WorkflowServiceRepeatActionAction,
		),
		Dao:      NewWorkflowDao(),
		registry: make(map[string]*Workflow),
	}
	result.AbstractService.Service = result
	return result
}
