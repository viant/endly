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
	"log"
	"path"
	"strings"
	"sync"
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
	context.Publish(model.NewModifiedStateEvent(variables, in, out))
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

func (s *Service) runAction(context *endly.Context, action *model.Action, process *model.Process) (response map[string]interface{}, err error) {
	var state = context.State()
	activity := model.NewActivity(context, action, state)
	defer func() {
		var resultKey = action.Name
		if resultKey == "" {
			resultKey = action.Action
		}
		if err != nil {
			err = fmt.Errorf("%v: %v", action.TagID, err)
		} else if len(response) > 0 {
			state.Put(resultKey, response)
			var variables = model.Variables{
				{
					Name:  resultKey,
					Value: response,
				},
			}
			_ = variables.Apply(state, state)
			context.Publish(model.NewModifiedStateEvent(variables, state, state))
		}
	}()
	var request interface{}
	state.Put("tagId", action.TagID)

	err = s.runNode(context, "action", process, action.AbstractNode, func(context *endly.Context, process *model.Process) (in, out data.Map, err error) {
		process.Push(activity)
		startEvent := s.Begin(context, activity)
		defer s.End(context)(startEvent, model.NewActivityEndEvent(activity))
		defer process.Pop()

		requestMap := toolbox.AsMap(activity.Request)
		if request, err = context.AsRequest(activity.Service, activity.Action, requestMap); err != nil {
			return nil, nil, err
		}
		err = endly.Run(context, request, activity.ServiceResponse)
		if err != nil {
			return nil, nil, err
		}

		_ = toolbox.DefaultConverter.AssignConverted(&activity.Response, activity.ServiceResponse.Response)
		response = activity.Response
		if runResponse, ok := activity.ServiceResponse.Response.(*RunResponse); ok {
			response = runResponse.Data
		}
		return response, state, err
	})
	return response, err
}

func (s *Service) runTask(context *endly.Context, process *model.Process, task *model.Task) (data.Map, error) {
	process.SetTask(task)
	var result = data.NewMap()
	var state = context.State()

	asyncGroup := &sync.WaitGroup{}
	var asyncError error
	asyncActions := task.AsyncActions()

	err := s.runNode(context, "task", process, task.AbstractNode, func(context *endly.Context, process *model.Process) (in, out data.Map, err error) {
		if task.TasksNode != nil && len(task.Tasks) > 0 {
			if err := s.runTasks(context, process, task.TasksNode); err != nil || len(task.Actions) == 0 {
				return state, result, err
			}
		}
		if len(asyncActions) > 0 {
			s.runAsyncActions(context, process, task, asyncActions, asyncGroup, &asyncError)
		}
		for i := 0; i < len(task.Actions); i++ {
			action := task.Actions[i]
			if action.Async {
				continue
			}
			if process.HasTagID && !process.TagIDs[action.TagID] {
				continue
			}

			var handler = func(action *model.Action) func() (interface{}, error) {
				return func() (interface{}, error) {
					var response, err = s.runAction(context, action, process)
					if err != nil {
						return nil, err
					}
					if len(response) > 0 {
						result[action.ID()] = response
					}
					return response, nil
				}
			}
			moveToNextTag, err := criteria.Evaluate(context, context.State(), action.Skip, "Skip", false)
			if err != nil {
				return nil, nil, err
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
				return nil, nil, err
			}
		}

		return state, result, nil
	})

	if len(asyncActions) > 0 {
		asyncGroup.Wait()
		if err == nil && asyncError != nil {
			err = asyncError
		}
	}
	state.Apply(result)
	return result, err
}

func (s *Service) runAsyncAction(parent, context *endly.Context, process *model.Process, action *model.Action, group *sync.WaitGroup) error {
	defer group.Done()
	events := context.MakeAsyncSafe()
	defer func() {
		for _, event := range events.Events {
			parent.Publish(event)
		}
	}()
	var result = make(map[string]interface{})
	var handler = func(action *model.Action) func() (interface{}, error) {
		return func() (interface{}, error) {
			var response, err = s.runAction(context, action, process)
			if err != nil {
				return nil, err
			}
			if len(response) > 0 {
				result[action.ID()] = response
			}
			return response, nil
		}
	}
	var extractable = make(map[string]interface{})
	err := action.Repeater.Run(s.AbstractService, "action", context, handler(action), extractable)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) runAsyncActions(context *endly.Context, process *model.Process, task *model.Task, asyncAction []*model.Action, group *sync.WaitGroup, asyncError *error) {
	if len(asyncAction) > 0 {
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
		if groupErr != nil {
			*asyncError = groupErr
		}
	}
}

func (s *Service) applyVariables(candidates interface{}, process *model.Process, in data.Map, context *endly.Context) error {
	variables, ok := candidates.(model.Variables)
	if !ok || len(variables) == 0 {
		return nil
	}

	var out = context.State()
	err := variables.Apply(in, out)
	s.addVariableEvent("Pipeline", variables, context, in, out)
	return err
}

func (s *Service) run(context *endly.Context, request *RunRequest) (response *RunResponse, err error) {
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

func (s *Service) publishParameters(request *RunRequest, context *endly.Context) map[string]interface{} {
	var state = context.State()
	params := buildParamsMap(request, context)
	if request.PublishParameters {
		for key, value := range params {
			state.Put(key, value)
		}
	}
	state.Put(paramsStateKey, params)
	return params
}

func (s *Service) getWorkflow(context *endly.Context, request *RunRequest) (*model.Workflow, error) {
	if request.workflow != nil {
		context.Publish(NewLoadedEvent(request.workflow))
		return request.workflow, nil
	}
	err := s.loadWorkflowIfNeeded(context, request)
	if err != nil {
		return nil, err
	}
	workflow, err := s.Workflow(request.Name)
	if err != nil {
		return nil, err
	}
	context.Publish(NewLoadedEvent(workflow))
	return workflow, err
}

func (s *Service) runWorkflow(upstreamContext *endly.Context, request *RunRequest) (response *RunResponse, err error) {
	response = &RunResponse{
		Data:      make(map[string]interface{}),
		SessionID: upstreamContext.SessionID,
	}

	s.enableLoggingIfNeeded(upstreamContext, request)
	workflow, err := s.getWorkflow(upstreamContext, request)
	if err != nil {
		return nil, err
	}

	defer Pop(upstreamContext)

	upstreamProcess := Last(upstreamContext)
	process := model.NewProcess(workflow.Source, workflow, upstreamProcess)
	process.AddTagIDs(strings.Split(request.TagIDs, ",")...)
	Push(upstreamContext, process)

	var workflowState = data.NewMap()
	upstreamState := upstreamContext.State()
	if request.StateKey != "" {
		if upstreamState.Has(request.StateKey) {
			log.Print("detected workflow state key: %v is taken by: %v, skiping consider stateKey customiztion", request.StateKey, upstreamState.Get(request.StateKey))
		}
		upstreamState.Put(request.StateKey, workflowState)
		defer func() {
			upstreamState.Delete(request.StateKey)
		}()
	}

	context := upstreamContext
	if !request.SharedState {
		context = upstreamContext.Clone()
	}
	params := s.publishParameters(request, context)
	workflowState.Put(paramsStateKey, params)
	if len(workflow.Data) > 0 {
		state := context.State()
		state.Put(dataStateKey, workflow.Data)
		workflowState.Put(dataStateKey, workflow.Data)
	}

	var state = context.State()
	upstreamTasks, hasUpstreamTasks := state.GetValue(tasksStateKey)
	restore := context.PublishAndRestore(toolbox.Pairs(
		neatly.OwnerURL, workflow.Source.URL,
		tasksStateKey, request.Tasks,
	))
	defer restore()

	context.Publish(NewInitEvent(request.Tasks, state))

	taskSelector := model.TasksSelector(request.Tasks)

	if !taskSelector.RunAll() {
		for _, task := range taskSelector.Tasks() {
			if !workflow.TasksNode.Has(task) {
				if hasUpstreamTasks && request.Tasks == toolbox.AsString(upstreamTasks) {
					taskSelector = model.TasksSelector("*")
				} else {
					return nil, fmt.Errorf("failed to lookup task: %v . %v", workflow.Name, task)
				}
			}
		}
	}

	filteredTasks := workflow.TasksNode.Select(taskSelector)
	if err != nil {
		return response, err
	}

	err = s.runNode(context, "workflow", process, workflow.AbstractNode, func(context *endly.Context, process *model.Process) (in, out data.Map, err error) {
		err = s.runTasks(context, process, filteredTasks)
		return state, response.Data, err
	})

	if len(response.Data) > 0 {
		for k, v := range response.Data {
			upstreamState.Put(k, v)
		}
	}
	return response, err
}

func (s *Service) runNode(context *endly.Context, nodeType string, process *model.Process, node *model.AbstractNode, runHandler func(context *endly.Context, process *model.Process) (in, out data.Map, err error)) error {
	if !process.CanRun() {
		return nil
	}
	var state = context.State()
	canRun, err := criteria.Evaluate(context, context.State(), node.When, fmt.Sprintf("%v.When", nodeType), true)
	if err != nil || !canRun {
		return err
	}
	err = node.Init.Apply(state, state)
	s.addVariableEvent(fmt.Sprintf("%v.Init", nodeType), node.Init, context, state, state)
	if err != nil {
		return err
	}
	in, out, err := runHandler(context, process)
	if err != nil {
		return err
	}
	if len(in) == 0 {
		in = data.NewMap()
	}
	err = node.Post.Apply(in, out)
	s.addVariableEvent(fmt.Sprintf("%v.Post", nodeType), node.Post, context, in, out)
	if err != nil {
		return err
	}
	s.Sleep(context, node.SleepTimeMs)
	return nil
}

func (s *Service) runDeferredTask(context *endly.Context, process *model.Process, parent *model.TasksNode) error {
	if parent.DeferredTask == "" {
		return nil
	}
	task, _ := parent.Task(parent.DeferredTask)
	_, err := s.runTask(context, process, task)
	return err
}

func (s *Service) runOnErrorTask(context *endly.Context, process *model.Process, parent *model.TasksNode, err error) error {
	if parent.OnErrorTask == "" {
		return err
	}
	if err != nil {
		process.Error = err.Error()
		if process.Activity != nil {
			process.Request = process.Activity.Request
			process.Response = process.Activity.Response
			process.TaskName = process.Task.Name
		}
		var state = context.State()
		var processErr = process.AsMap()
		state.Put("error", processErr)
		processErr = toolbox.DeleteEmptyKeys(processErr)
		errorJSON, err := toolbox.AsIndentJSONText(processErr)
		state.Put("errorJSON", errorJSON)
		task, e := parent.Task(parent.OnErrorTask)
		if e != nil {
			return fmt.Errorf("failed to catch: %v, %v", err, e)
		}
		_, err = s.runTask(context, process, task)
		return err
	}
	return err
}

func (s *Service) runTasks(context *endly.Context, process *model.Process, tasks *model.TasksNode) (err error) {
	defer func() {
		e := s.runDeferredTask(context, process, tasks)
		if err == nil {
			err = e
		}
	}()
	for _, task := range tasks.Tasks {
		if task.Name == tasks.OnErrorTask || task.Name == tasks.DeferredTask {
			continue
		}
		if process.IsTerminated() {
			break
		}
		if _, err = s.runTask(context, process, task); err != nil {
			err = s.runOnErrorTask(context, process, tasks, err)
		}
		if err != nil {
			return err
		}
	}
	var scheduledTask = process.Scheduled
	if scheduledTask != nil {
		process.Scheduled = nil
		err = s.runTasks(context, process, &model.TasksNode{Tasks: []*model.Task{scheduledTask}})
	}
	return err
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
	var result = actionRequest.NewAction()

	if activity != nil {
		result.NeatlyTag = activity.NeatlyTag
		result.Name = activity.Action
		if result.AbstractNode.Description == "" {
			result.AbstractNode.Description = activity.Description
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
			return s.runTask(context, process, task)
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
  "Tasks": "start"
}`

	inlineWorkflowServiceRunExample = `{
	"Params": {
		"app": "myapp",
		"appTarget": {
			"Credentials": "localhost",
			"URL": "ssh://127.0.0.1/"
		},
		"buildTarget": {
			"Credentials": "localhost",
			"URL": "ssh://127.0.0.1/"
		},
		"commands": [
			"export GOPATH=/tmp/go",
			"go get -u -v github.com/viant/endly/bootstrap",
			"cd ${buildPath}app",
			"go build -o myapp",
			"chmod +x myapp"
		],
		"download": [
			{
				"Key": "${buildPath}/app/myapp",
				"Value": "$releasePath"
			}
		],
		"origin": [
			{
				"Key": "URL",
				"Value": "./../"
			}
		],
		"sdk": "go:1.9",
		"target": {
			"Credentials": "localhost",
			"URL": "ssh://127.0.0.1/"
		}
	},
	"PublishParameters": true,
	"Tasks": "*",
	"URL": "app/build.csv"
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
					Description: "run external workflow",
					Data:        workflowServiceRunExample,
				},
				{
					Description: "run inline workflow",
					Data:        inlineWorkflowServiceRunExample,
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
