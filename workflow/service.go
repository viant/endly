package workflow

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/viant/endly"
	"github.com/viant/endly/criteria"
	"github.com/viant/endly/model"
	"github.com/viant/endly/msg"
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
)

//Service represents a workflow service.
type Service struct {
	*endly.AbstractService
	Dao       *Dao
	registry  map[string]*model.Workflow
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
func (s *Service) Register(workflow *model.Workflow) error {
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
func (s *Service) Workflow(name string) (*model.Workflow, error) {
	s.Lock()
	defer s.Unlock()
	if result, found := s.registry[name]; found {
		return result, nil
	}
	return nil, fmt.Errorf("failed to lookup workflow: %v", name)
}

func (s *Service) addVariableEvent(name string, variables model.Variables, context *endly.Context, in, out data.Map) {
	if len(variables) == 0 {
		return
	}
	context.Publish(NewModifiedStateEvent(variables, in, out))
}

func getURLs(URL string) []string {
	selector := model.WorkflowSelector(URL)
	workflowName := selector.Name()
	workflowFilename := fmt.Sprintf("%v.csv", workflowName)
	dedicatedFolderURL := strings.Replace(URL, workflowFilename, fmt.Sprintf("%v/%v", workflowName, workflowFilename), 1)
	return []string{
		URL,
		dedicatedFolderURL,
	}
}

//GetResource returns workflow resource
func GetResource(dao *Dao, state data.Map, URL string) *url.Resource {
	for _, candidate := range getURLs(URL) {
		resource := url.NewResource(candidate)
		storageService, err := storage.NewServiceForURL(resource.URL, "")
		if err != nil {
			return nil
		}
		exists, _ := storageService.Exists(resource.URL)
		if exists {
			return resource
		}
	}
	if strings.Contains(URL, ":/") || strings.HasPrefix(URL, "/") {
		return nil
	}
	//Lookup shared workflow
	for _, candidate := range getURLs(URL) {
		resource, err := dao.NewRepoResource(state, fmt.Sprintf("workflow/%v", candidate))
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

func (s *Service) loadWorkflowIfNeeded(context *endly.Context, request *RunRequest) (err error) {
	if !s.HasWorkflow(request.Name) {
		resource := GetResource(s.Dao, context.State(), request.URL)
		if resource == nil {
			return fmt.Errorf("unable to locate workflow: %v, %v", request.Name, request.URL)
		}
		if _, err := s.loadWorkflow(context, &LoadRequest{Source: resource}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) getServiceRequest(context *endly.Context, activity *model.Activity) (endly.Service, interface{}, error) {
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

func (s *Service) runAction(context *endly.Context, action *model.Action, process *model.Process) (response interface{}, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("%v: %v", action.TagID, err)
		}
	}()
	if !process.CanRun() {
		return nil, nil
	}
	var state = context.State()
	activity := model.NewActivity(context, action, state)
	process.Push(activity)

	startEvent := s.Begin(context, activity)

	defer s.End(context)(startEvent, model.NewActivityEndEvent(activity))
	defer process.Pop()

	var canRun bool
	canRun, err = criteria.Evaluate(context, context.State(), action.When, "action.When", true)
	if err != nil || !canRun {
		activity.Ineligible = true
		return nil, err
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

func (s *Service) injectTagIDsIfNeeded(action *model.ServiceRequest, tagIDs map[string]bool) {
	if action.Service != "workflow" || action.Action != "runWorkflow" {
		return
	}
	requestMap := toolbox.AsMap(action.Request)
	requestMap["TagIDs"] = strings.Join(toolbox.MapKeysToStringSlice(tagIDs), ",")
}

func (s *Service) runTask(context *endly.Context, process *model.Process, tagIDs map[string]bool, task *model.Task) (data.Map, error) {
	if !process.CanRun() {
		return nil, nil
	}
	process.TaskName = task.Name
	var startTime = time.Now()
	var state = context.State()
	canRun, err := criteria.Evaluate(context, context.State(), task.When, "task.When", true)
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

	var asyncActions = make([]*model.Action, 0)
	for i := 0; i < len(task.Actions); i++ {
		action := task.Actions[i]
		if hasTagIDs {
			s.injectTagIDsIfNeeded(action.ServiceRequest, tagIDs)
		}
		if filterTagIDs && !tagIDs[action.TagID] {
			continue
		}
		if action.Async {
			asyncActions = append(asyncActions, task.Actions[i])
			continue
		}

		var handler = func(action *model.Action) func() (interface{}, error) {
			return func() (interface{}, error) {
				return s.runAction(context, action, process)
			}
		}
		moveToNextTag, err := criteria.Evaluate(context, context.State(), action.Skip, "TagIdSkipCriteria", false)
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
		err = action.Repeater.Run(s.AbstractService, "action", context, handler(task.Actions[i]), extractable)
		if err != nil {
			return nil, err
		}
	}
	err = s.runAsyncActions(context, process, task, asyncActions)
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

func (s *Service) applyRemainingTaskSpentIfNeeded(context *endly.Context, task *model.Task, startTime time.Time) {
	if task.TimeSpentMs > 0 {
		var elapsed = (time.Now().UnixNano() - startTime.UnixNano()) / int64(time.Millisecond)
		var remainingExecutionTime = time.Duration(task.TimeSpentMs - int(elapsed))
		s.Sleep(context, int(remainingExecutionTime))
	}
}

func (s *Service) runAsyncAction(parent, context *endly.Context, process *model.Process, action *model.Action, group *sync.WaitGroup) error {
	defer group.Done()

	events := context.MakeAsyncSafe()
	defer func() {
		for _, event := range events.Events {
			parent.Publish(event)
		}
	}()
	if _, err := s.runAction(context, action, process); err != nil {
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

func (s *Service) runAsyncActions(context *endly.Context, process *model.Process, task *model.Task, asyncAction []*model.Action) error {
	if len(asyncAction) > 0 {
		group := &sync.WaitGroup{}
		group.Add(len(asyncAction))
		var groupErr error
		for i := range asyncAction {
			context.Publish(NewAsyncEvent(asyncAction[i]))
			go func(action *model.Action, actionContext *endly.Context) {
				if err := s.runAsyncAction(context, actionContext, process, action, group); err != nil {
					groupErr = err
				}
			}(asyncAction[i], context.Clone())
		}
		group.Wait()
		if groupErr != nil {
			return groupErr
		}
	}
	return nil
}

func (s *Service) runPipeline(context *endly.Context, pipeline *model.Pipeline, response *RunResponse, process *model.Process) (err error) {
	context.Publish(NewPipelineEvent(pipeline))
	var state = context.State()
	last := LastWorkflow(context)
	if len(pipeline.Pipelines) > 0 {
		defer Pop(context)
		process := model.NewProcess(process.Source, nil, pipeline)
		Push(context, process)
		return s.traversePipelines(pipeline.Pipelines, context, response, process)
	}

	activity := pipeline.NewActivity(context)
	if activity.Caller == "" && last != nil {
		lastActivity := last.Last()
		if lastActivity != nil {
			activity.Caller = last.Activity.Caller
			activity.TagID = last.Activity.TagID
			activity.Tag = last.Activity.Tag
		}
	}

	context.Publish(activity)
	process.Push(activity)
	defer process.Pop()
	runResponse := &RunResponse{
		Data: make(map[string]interface{}),
	}

	var canRun bool
	canRun, err = criteria.Evaluate(context, context.State(), pipeline.When, "pipeline.When", true)
	if err != nil || !canRun {
		activity.Ineligible = true
		return err
	}

	defer func() {
		context.Publish(model.NewActivityEndEvent(runResponse))
		if len(runResponse.Data) > 0 {
			if pipeline.Post != nil {
				s.applyVariables(pipeline.Post, process, runResponse.Data, context)
			}
			for k, v := range runResponse.Data {
				state.Put(k, v)
			}
			response.Data[pipeline.Name] = runResponse.Data
		}
	}()

	if pipeline.Init != nil {
		s.applyVariables(pipeline.Init, process, state, context)
	}

	var request = toolbox.AsMap(state.Expand(activity.Request))
	if pipeline.Workflow != "" {
		runRequest := NewRunRequest(pipeline.Workflow, pipeline.Params, true)
		if err = endly.Run(context, runRequest, runResponse); err != nil {
			return err
		}
	} else if pipeline.Action != "" {
		actionSelector := model.ActionSelector(pipeline.Action)
		var request, err = context.AsRequest(actionSelector.Service(), actionSelector.Action(), request)
		if err != nil {
			return err
		}
		if err = endly.Run(context, request, &runResponse.Data); err != nil {
			return err
		}
	}
	return err
}

func (s *Service) traversePipelines(pipelines model.Pipelines, context *endly.Context, response *RunResponse, process *model.Process) error {
	for i := range pipelines {
		if err := s.runPipeline(context, pipelines[i], response, process); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) applyVariables(candidates interface{}, process *model.Process, in data.Map, context *endly.Context) error {
	variables, ok := candidates.(model.Variables)
	if !ok {
		return nil
	}
	var out = context.State()
	err := variables.Apply(in, out)
	s.addVariableEvent("Pipeline", variables, context, in, out)
	return err
}

func (s *Service) pipeline(context *endly.Context, request *RunRequest) (*RunResponse, error) {

	var response = &RunResponse{
		Data: make(map[string]interface{}),
	}
	if len(request.Pipelines) == 0 {
		return response, nil
	}

	defer Pop(context)
	process := model.NewProcess(request.Source, nil, request.Pipelines[0])
	Push(context, process)

	s.publishParameters(request, context)
	var state = context.State()
	if request.Inline.Init != nil {
		if err := s.applyVariables(request.Inline.Init, process, state, context); err != nil {
			return nil, err
		}
	}

	response, err := response, s.traversePipelines(request.Inline.Pipelines, context, response, process)
	if request.Inline.Post != nil && response != nil {
		if err := s.applyVariables(request.Inline.Post, process, response.Data, context); err != nil {
			return nil, err
		}
	}
	return response, err
}

func (s *Service) pipelineWorkflowAsyncInNeeded(context *endly.Context, request *RunRequest) (*RunResponse, error) {

	s.enableLoggingIfNeeded(context, request)
	if request.Async {
		context.Wait.Add(1)
		go func() {
			defer context.Wait.Done()
			_, err := s.pipeline(context, request)
			if err != nil {
				context.Publish(msg.NewErrorEvent(fmt.Sprintf("%v", err)))
			}

		}()
		return &RunResponse{}, nil
	}
	return s.pipeline(context, request)
}

func (s *Service) run(context *endly.Context, request *RunRequest) (response *RunResponse, err error) {
	if request.Inline != nil {
		return s.pipelineWorkflowAsyncInNeeded(context, request)
	}
	return s.runWorkflowAsyncIfNeeded(context, request)
}

func (s *Service) runWorkflowAsyncIfNeeded(context *endly.Context, request *RunRequest) (response *RunResponse, err error) {
	if request.Async {
		context.Wait.Add(1)
		go func() {
			defer context.Publish(NewEndEvent(context.SessionID))
			defer context.Wait.Done()
			_, err = s.runWorkflow(context, request)
			if err != nil {
				context.Publish(msg.NewErrorEvent(fmt.Sprintf("%v", err)))
			}
		}()
		return &RunResponse{}, nil
	}
	defer context.Publish(NewEndEvent(context.SessionID))
	return s.runWorkflow(context, request)
}

func (s *Service) enableLoggingIfNeeded(context *endly.Context, request *RunRequest) {
	if request.EnableLogging && !context.HasLogger {
		var logDirectory = path.Join(request.LogDirectory, context.SessionID)
		logger := NewLogger(logDirectory, context.Listener)
		context.Listener = logger.AsEventListener()
	}
}

func (s *Service) publishParameters(request *RunRequest, context *endly.Context) {
	var state = context.State()
	params := buildParamsMap(request, context)
	if request.PublishParameters {
		for key, value := range params {
			state.Put(key, value)
		}
	}
	state.Put(paramsStateKey, params)
}

func (s *Service) runWorkflow(upstreamContext *endly.Context, request *RunRequest) (response *RunResponse, err error) {
	response = &RunResponse{
		Data:      make(map[string]interface{}),
		SessionID: upstreamContext.SessionID,
	}

	s.enableLoggingIfNeeded(upstreamContext, request)
	err = s.loadWorkflowIfNeeded(upstreamContext, request)
	if err != nil {
		return response, err
	}
	workflow, err := s.Workflow(request.Name)
	if err != nil {
		return response, err
	}
	upstreamContext.Publish(NewLoadedEvent(workflow))
	defer Pop(upstreamContext)
	process := model.NewProcess(workflow.Source, workflow, nil)
	Push(upstreamContext, process)

	context := upstreamContext.Clone()
	var state = context.State()

	var workflowData = data.Map(workflow.Data)
	state.Put(neatly.OwnerURL, workflow.Source.URL)
	state.Put(dataStateKey, workflowData)
	state.Put(tasksStateKey, request.Tasks)
	s.publishParameters(request, context)

	err = workflow.Init.Apply(state, state)
	s.addVariableEvent("Workflow.Init", workflow.Init, context, state, state)
	if err != nil {
		return response, err
	}
	context.Publish(NewInitEvent(request.Tasks, state))
	filteredTasks, err := workflow.Select(model.TasksSelector(request.Tasks))
	if err != nil {
		return response, err
	}

	var tagIDs = make(map[string]bool)
	for _, tagID := range strings.Split(request.TagIDs, ",") {
		tagIDs[tagID] = true
	}

	defer s.runWorkflowDeferTaskIfNeeded(context, process)
	err = s.runWorkflowTasks(context, process, tagIDs, filteredTasks...)
	err = s.runOnErrorTaskIfNeeded(context, process, err)
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

func (s *Service) runWorkflowDeferTaskIfNeeded(context *endly.Context, process *model.Process) {
	if process.Workflow.DeferTask == "" {
		return
	}
	task, _ := process.Workflow.Task(process.Workflow.DeferTask)
	_ = s.runWorkflowTasks(context, process, nil, task)
}

func (s *Service) runOnErrorTaskIfNeeded(context *endly.Context, process *model.Process, err error) error {
	if err != nil {
		if process.Workflow.OnErrorTask == "" {
			return err
		}
		process.Error = err.Error()

		var errorMap = toolbox.AsMap(process.ExecutionError)
		activity := process.Activity
		if activity != nil && activity.Request != nil {
			errorMap["ServiceRequest"], _ = toolbox.AsJSONText(activity.Request)
		}
		if activity != nil && activity.Response != nil {
			errorMap["Response"], _ = toolbox.AsJSONText(activity.Response)
		}
		var state = context.State()
		state.Put("error", errorMap)
		task, _ := process.Workflow.Task(process.Workflow.OnErrorTask)
		err = s.runWorkflowTasks(context, process, nil, task)
	}
	return err
}

func (s *Service) runWorkflowTasks(context *endly.Context, process *model.Process, tagIDs map[string]bool, tasks ...*model.Task) error {
	for _, task := range tasks {
		if process.IsTerminated() {
			break
		}
		if _, err := s.runTask(context, process, tagIDs, task); err != nil {
			return err
		}
	}
	var scheduledTask = process.Scheduled
	if scheduledTask != nil {
		process.Scheduled = nil
		return s.runWorkflowTasks(context, process, tagIDs, scheduledTask)
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
	process := Last(context)
	if process != nil {
		process.Terminate()
	}
	return &ExitResponse{}, nil
}

func (s *Service) runGoto(context *endly.Context, request *GotoRequest) (GotoResponse, error) {
	var response interface{}
	process := Last(context)
	if process == nil {
		err := fmt.Errorf("no active workflow")
		return nil, err
	}
	var task *model.Task
	task, err := process.Workflow.Task(request.Task)
	if err == nil {
		process.Scheduled = task
	}
	return response, err
}

func getServiceActivity(context *endly.Context) *model.Activity {
	process := Last(context)
	if process == nil {
		return nil
	}
	return process.Activity
}

func getServiceAction(context *endly.Context, actionRequest *model.ServiceRequest) *model.Action {
	activity := getServiceActivity(context)
	var result = &model.Action{
		ServiceRequest: actionRequest,
		NeatlyTag:      &model.NeatlyTag{},
	}
	if activity != nil {
		result.NeatlyTag = activity.NeatlyTag
		result.Name = activity.Action
		if result.Description == "" {
			result.Description = activity.Description
		}
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
	process := LastWorkflow(context)
	if process == nil {
		return nil, errors.New("no active workflow")
	}
	var response interface{}
	var source = getSwitchSource(context, request.SourceKey)
	matched := request.Match(source)
	if matched != nil {
		if matched.Task != "" {
			task, err := process.Workflow.Task(matched.Task)
			if err != nil {
				return nil, err
			}
			return s.runTask(context, process, nil, task)
		}
		serviceAction := getServiceAction(context, matched.ServiceRequest)
		return s.runAction(context, serviceAction, process)

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
  "TasksSelector": "start"
}`

	workflowServiceSwitchExample = `{
  "SourceKey": "instanceState",
  "Cases": [
    {
      "Service": "aws/ec2",
      "Action": "call",
      "ServiceRequest": {
        "Credentials": "${env.HOME}/.secret/aws-west.json",
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
	s.AbstractService.Register(&endly.Route{
		Action: "run",
		RequestInfo: &endly.ActionInfo{
			Description: "runWorkflow workflow",
			Examples: []*endly.UseCase{
				{
					Description: "runWorkflow workflow",
					Data:        workflowServiceRunExample,
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
				return s.run(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.AbstractService.Register(&endly.Route{
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

	s.AbstractService.Register(&endly.Route{
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

	s.AbstractService.Register(&endly.Route{
		Action: "switch",
		RequestInfo: &endly.ActionInfo{
			Description: "select action or task for matched case value",
			Examples: []*endly.UseCase{
				{
					Description: "switch case",
					Data:        workflowServiceSwitchExample,
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

	s.AbstractService.Register(&endly.Route{
		Action: "goto",
		RequestInfo: &endly.ActionInfo{
			Description: "goto task",
			Examples: []*endly.UseCase{
				{
					Description: "goto",
					Data:        workflowServiceGotoExample,
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

	s.AbstractService.Register(&endly.Route{
		Action: "exit",
		RequestInfo: &endly.ActionInfo{
			Description: "exit current workflow",
			Examples: []*endly.UseCase{
				{
					Description: "exit",
					Data:        workflowServiceExitExample,
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

	s.AbstractService.Register(&endly.Route{
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

	s.AbstractService.Register(&endly.Route{
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

	s.AbstractService.Register(&endly.Route{
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
		registry:        make(map[string]*model.Workflow),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
