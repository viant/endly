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

	//WorkflowServiceSwitchAction represents workflow switch action
	WorkflowServiceSwitchAction = "switch"

	//WorkflowServiceRegisterAction represents workflow register action
	WorkflowServiceRegisterAction = "register"

	//WorkflowServiceLoadAction represents workflow load action
	WorkflowServiceLoadAction = "load"

	//WorkflowServiceExitAction represents exit action
	WorkflowServiceExitAction = "exit"

	//WorkflowServiceGotoAction represents goto task action
	WorkflowServiceGotoAction = "goto"

	//WorkflowServiceActivityKey activity key
	WorkflowServiceActivityKey = "activity"

	workflowCaller = "Worfklow"

	workflowError = "error"
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

func (s *workflowService) runAction(context *Context, action *ServiceAction, workflow *WorkflowControl) (interface{}, error) {
	if !workflow.CanRun() {
		return nil, nil
	}
	workflow.ActionRequest = action.ActionRequest
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
	workflow.Request = serviceRequest
	serviceResponse := service.Run(context, serviceRequest)

	workflow.Response = serviceResponse
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
	return response, nil
}

func (s *workflowService) runTask(context *Context, workflow *WorkflowControl, task *WorkflowTask) (data.Map, error) {
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
			var asyncEvent = NewAsyncServiceActionEvent(workflow.Name, context.Expand(task.Name), context.Expand(action.Description), action.Service, action.Action, action.TagID)
			AddEvent(context, asyncEvent, Pairs("value", asyncEvent))
			continue
		}

		var handler = func() (interface{}, error) {
			result, err := s.runAction(context, action, workflow)
			if err != nil {
				return nil, fmt.Errorf("failed to run action:%v %v", action.Tag, err)
			}
			return result, nil
		}

		var extractable = make(map[string]string)
		repeatable := action.Repeatable.Get()
		err = repeatable.Run(s.AbstractService, workflowCaller, context, handler, extractable)
		if err != nil {
			return nil, err
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
	control := upstreamContext.Workflows.Push(workflow)
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
	AddEvent(context, "State.Init", Pairs("state", state.AsEncodableMap(), "tasks", request.Tasks), Debug)
	filteredTasks, err := workflow.FilterTasks(request.Tasks)
	if err != nil {
		return nil, err
	}

	defer s.runWorkflowDeferTaskIfNeeded(context, control)
	err = s.runWorkflowTasks(context, control, filteredTasks...)
	err = s.runOnErrorTaskIfNeeded(context, control, err)
	if err != nil {
		return nil, err
	}

	workflow.Post.Apply(state, response.Data) //context -> workflow output
	s.addVariableEvent("Workflow.Post", workflow.Post, context, state)

	if workflow.SleepTimeMs > 0 {
		s.Sleep(context, workflow.SleepTimeMs)
	}
	return response, nil
}

func (s *workflowService) runWorkflowDeferTaskIfNeeded(context *Context, workflow *WorkflowControl) {
	if workflow.DeferTask == "" {
		return
	}
	task, _ := workflow.Task(workflow.DeferTask)
	_ = s.runWorkflowTasks(context, workflow, task)
}

func (s *workflowService) runOnErrorTaskIfNeeded(context *Context, workflow *WorkflowControl, err error) error {
	if err != nil {
		if workflow.OnErrorTask == "" {
			return err
		}
		workflow.Error = err.Error()

		var errorMap = toolbox.AsMap(workflow.WorkflowError)
		if workflow.WorkflowError.Request != nil {
			errorMap["Request"], _ = toolbox.AsJSONText(workflow.WorkflowError.Request)
		}
		if workflow.WorkflowError.Response != nil {
			errorMap["Response"], _ = toolbox.AsJSONText(workflow.WorkflowError.Response)
		}
		context.state.Put(workflowError, errorMap)
		task, _ := workflow.Task(workflow.OnErrorTask)
		err = s.runWorkflowTasks(context, workflow, task)
	}
	return err
}

func (s *workflowService) runWorkflowTasks(context *Context, workflow *WorkflowControl, tasks ...*WorkflowTask) error {
	for _, task := range tasks {
		if workflow.IsTerminated() {
			break
		}
		if _, err := s.runTask(context, workflow, task); err != nil {
			return err
		}
	}
	var scheduledTask = workflow.ScheduledTask
	if scheduledTask != nil {
		workflow.ScheduledTask = nil
		return s.runWorkflowTasks(context, workflow, scheduledTask)
	}
	return nil
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

	case *WorkflowSwitchActionRequest:
		response.Response, err = s.runSwitch(context, actualRequest)
		errorTemplate = "%v"

	case *WorkflowExitRequest:
		control := context.Workflows.LastControl()
		if control != nil {
			control.Terminate()
		}
		response.Response = &WorkflowExitResponse{}
	case *WorkflowGotoRequest:
		response.Response, err = s.runGoto(context, actualRequest)
	default:
		err = fmt.Errorf("unsupported request type: %T", request)
	}

	if err != nil {
		response.Status = "error"
		response.Error = fmt.Sprintf(errorTemplate, err)
	}

	return response
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

func (s *workflowService) NewRequest(action string) (interface{}, error) {
	switch action {
	case WorkflowServiceRunAction:
		return &WorkflowRunRequest{}, nil
	case WorkflowServiceRegisterAction:
		return &WorkflowRegisterRequest{}, nil
	case WorkflowServiceLoadAction:
		return &WorkflowLoadRequest{}, nil
	case WorkflowServiceSwitchAction:
		return &WorkflowSwitchActionRequest{}, nil
	case WorkflowServiceExitAction:
		return &WorkflowExitRequest{}, nil
	case WorkflowServiceGotoAction:
		return &WorkflowGotoRequest{}, nil
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

func getSwitchSource(context *Context, sourceKey string) interface{} {
	sourceKey = context.Expand(sourceKey)
	var state = context.state
	var result = state.Get(sourceKey)
	if result == nil {
		return nil
	}
	return toolbox.DereferenceValue(result)
}

func (s *workflowService) runSwitch(context *Context, request *WorkflowSwitchActionRequest) (WorkflowSwitchResponse, error) {
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
			return s.runTask(context, workflow, task)
		}
		serviceAction := getServiceAction(context.state, matched.ActionRequest)
		return s.runAction(context, serviceAction, workflow)

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
			WorkflowServiceSwitchAction,
			WorkflowServiceExitAction,
			WorkflowServiceGotoAction,
		),
		Dao:      NewWorkflowDao(),
		registry: make(map[string]*Workflow),
	}
	result.AbstractService.Service = result
	return result
}
