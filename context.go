package endly

import (
	"fmt"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/storage"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
	"sync/atomic"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
)

//TODO Execution detail Tracking of all run (time taken, request, response)

var converter = toolbox.NewColumnConverter("yyyy-MM-dd HH:ss")

type StateKey *data.Map

var serviceManagerKey = (*manager)(nil)
var deferFunctionsKey = (*[]func())(nil)
var workflowKey = (*Workflow)(nil)

type Context struct {
	SessionId     string
	state         data.Map
	toolbox.Context
	Events        *Events
	EventLogger   *EventLogger
	workflowStack *[]*Workflow
	cloned        []*Context
	closed        int32
}

func (c *Context) IsClosed() bool {
	return atomic.LoadInt32(&c.closed) == 1
}

func (c *Context) PushWorkflow(workflow *Workflow) {
	*c.workflowStack = append(*c.workflowStack, workflow)
}

func (c *Context) ShiftWorkflow() *Workflow {
	var result = (*c.workflowStack)[0]
	(*c.workflowStack) = (*c.workflowStack)[1:]
	return result
}

func (c *Context) CurrentWorkflow() *Workflow {
	if c.workflowStack == nil {
		return nil
	}
	var workflowCount = len(*c.workflowStack)
	if workflowCount == 0 {
		return nil
	}
	return (*c.workflowStack)[workflowCount-1]
}

func reportError(err error) error {
	fileName, funcName, line := toolbox.CallerInfo(4)
	return fmt.Errorf("%v at %v:%v -> %v", err, fileName, line, funcName)
}

func (c *Context) Clone() *Context {
	if len(c.cloned) == 0 {
		c.cloned = make([]*Context, 0)
	}
	result := &Context{}
	result.Context = c.Context.Clone()
	result.Events = c.Events
	result.state = NewDefaultState()
	result.state.Apply(c.state)
	result.SessionId = c.SessionId
	result.workflowStack = c.workflowStack
	result.EventLogger = c.EventLogger
	c.cloned = append(c.cloned, result)
	return result
}

func (c *Context) parentURLCandidates() []string {
	var result = make([]string, 0)
	if workflow := c.Workflow(); workflow != nil && workflow.Source != nil {
		baseURL, _ := toolbox.URLSplit(workflow.Source.URL)
		result = append(result, baseURL)
	}
	currentDirectory, err := os.Getwd()
	if err == nil {
		result = append(result, toolbox.FileSchema+currentDirectory)
	}

	return result
}

func (c *Context) ExpandResource(resource *url.Resource) (*url.Resource, error) {
	if resource == nil {
		return nil, reportError(fmt.Errorf("Resource was empty"))
	}
	if resource.URL == "" {
		return nil, reportError(fmt.Errorf("URL was empty"))
	}

	if !strings.Contains(resource.URL, "://") {
		for _, parentCandidate := range c.parentURLCandidates() {
			service, err := storage.NewServiceForURL(parentCandidate, "")
			if err != nil {
				continue
			}
			var candidateURL = toolbox.URLPathJoin(parentCandidate, resource.URL)
			if exists, err := service.Exists(candidateURL); exists && err == nil {
				resource.URL = candidateURL
			}
		}
	}
	var result = url.NewResource(c.Expand(resource.URL), c.Expand(resource.Credential))
	if result.ParsedURL == nil {
		return nil, fmt.Errorf("Failed to parse URL %v", result.URL)
	}
	result.Name = c.Expand(resource.Name)
	result.Version = resource.Version
	result.Type = c.Expand(resource.Type)
	result.Cache = c.Expand(resource.Cache)
	result.CacheExpiryMs = resource.CacheExpiryMs
	return result, nil
}

func (c *Context) Manager() (Manager, error) {
	var manager = &manager{}
	if !c.GetInto(serviceManagerKey, &manager) {
		return nil, reportError(fmt.Errorf("Failed to lookup Manager"))
	}
	return manager, nil
}

func (c *Context) Sessions() ClientSessions {
	var result *ClientSessions
	if !c.Contains(clientSessionKey) {
		var sessions ClientSessions = make(map[string]*ClientSession)
		result = &sessions
		c.Put(clientSessionKey, result)
	} else {
		c.GetInto(clientSessionKey, &result)
	}
	return *result
}

func (c *Context) Service(name string) (Service, error) {
	manager, err := c.Manager()
	if err != nil {
		return nil, err
	}
	return manager.Service(name)
}

func (c *Context) Deffer(functions ...func()) []func() {
	var result *[]func()
	if !c.Contains(deferFunctionsKey) {
		var functions = make([]func(), 0)
		result = &functions
		c.Put(deferFunctionsKey, result)
	} else {
		c.GetInto(deferFunctionsKey, &result)
	}

	*result = append(*result, functions...)
	c.Put(deferFunctionsKey, &result)
	return *result
}

func (c *Context) State() data.Map {
	if c.state == nil {
		c.state = NewDefaultState()
	}
	return c.state
}

func (c *Context) SetState(state data.Map) {
	c.state = state
}

func (c *Context) Workflow() *Workflow {
	var result *Workflow
	if !c.Contains(workflowKey) {
		return nil
	} else {
		c.GetInto(workflowKey, &result)
	}
	return result
}

func (c *Context) OperatingSystem(sessionName string) *OperatingSystem {
	var sessions = c.Sessions()
	if session, has := sessions[sessionName]; has {
		return session.OperatingSystem
	}
	return nil
}

func (c *Context) ExecuteAsSuperUser(target *url.Resource, command *ManagedCommand) (*CommandInfo, error) {
	superUserRequest := SuperUserCommandRequest{
		Target:        target,
		MangedCommand: command,
	}
	request, err := superUserRequest.AsCommandRequest(c)
	if err != nil {
		return nil, err
	}
	return c.Execute(target, request.ManagedCommand)
}

func (c *Context) Execute(target *url.Resource, command interface{}) (*CommandInfo, error) {
	if command == nil {
		return nil, nil
	}
	var commandRequest *ManagedCommandRequest
	switch actualCommand := command.(type) {
	case *CommandRequest:
		actualCommand.Target = target
		commandRequest = actualCommand.AsManagedCommandRequest()
	case *ManagedCommand:
		commandRequest = NewCommandRequest(target, actualCommand)
	case string:
		request := CommandRequest{
			Target:   target,
			Commands: []string{actualCommand},
		}
		commandRequest = request.AsManagedCommandRequest()
	case []string:
		request := CommandRequest{
			Target:   target,
			Commands: actualCommand,
		}
		commandRequest = request.AsManagedCommandRequest()

	default:
		return nil, fmt.Errorf("Unsupported command: %T", command)
	}
	execService, err := c.Service(ExecServiceId)
	if err != nil {
		return nil, err
	}
	response := execService.Run(c, commandRequest)
	if response.Error != "" {
		return nil, errors.New(response.Error)
	}
	if commandResult, ok := response.Response.(*CommandInfo); ok {
		return commandResult, nil
	}
	return nil, nil
}

func (c *Context) Copy(expand bool, source, target *url.Resource) (interface{}, error) {
	return c.Transfer([]*Transfer{{
		Source: source,
		Target: target,
		Expand: expand}}...)
}

func (c *Context) Transfer(transfers ...*Transfer) (interface{}, error) {
	if transfers == nil {
		return nil, nil
	}
	transferService, err := c.Service(TransferServiceId)
	if err != nil {
		return nil, err
	}
	response := transferService.Run(c, &TransferCopyRequest{Transfers: transfers})
	if response.Error != "" {
		return nil, errors.New(response.Error)
	}
	return nil, nil
}

func (c *Context) Expand(text string) string {
	state := c.State()
	return state.ExpandAsText(text)
}

func (c *Context) AsRequest(serviceName, requestName string, source map[string]interface{}) (interface{}, error) {
	service, err := c.Service(serviceName)
	if err != nil {
		return nil, err
	}
	request, err := service.NewRequest(requestName)
	if err != nil {
		return nil, err
	}
	err = converter.AssignConverted(request, source)
	return request, err
}

func (c *Context) Close() {
	atomic.StoreInt32(&c.closed, 1)
	for _, context := range c.cloned {
		context.Close()
	}
	for _, function := range c.Deffer() {
		function()
	}
}

func NewDefaultState() data.Map {
	var result = data.NewMap()
	var now = time.Now()
	source := rand.NewSource(now.UnixNano())
	result.Put("endlyURL", "http://github.com/viant/endly")
	result.Put("rand", source.Int63())
	result.Put("date", now.Format(toolbox.DateFormatToLayout("yyyy-MM-dd")))
	result.Put("time", now.Format(toolbox.DateFormatToLayout("yyyy-MM-dd hh:mm:ss")))
	result.Put("ts", now.Format(toolbox.DateFormatToLayout("yyyyMMddhhmmSSS")))

	result.Put("tmpDir", func(key string) interface{} {
		tempPath := path.Join(os.TempDir(), key)
		exec.Command("mkdir -p " + tempPath)
		return tempPath
	})

	var cachedUUID uuid.UUID
	result.Put("uuid", func(key string) interface{} {
		if key == "next" {
			cachedUUID = uuid.NewV4()
		}
		if len(cachedUUID) > 0 {
			return cachedUUID.String()
		}
		return ""
	})

	result.Put("timestamp", func(key string) interface{} {
		var timeDiffProvider = toolbox.NewTimeDiffProvider()
		switch key {
		case "now":
			result, _ := timeDiffProvider.Get(nil, "now", 0, "day", "timestamp")
			return result
		case "tomorrow":
			result, _ := timeDiffProvider.Get(nil, "now", 1, "day", "timestamp")
			return result
		case "yesterday":
			result, _ := timeDiffProvider.Get(nil, "now", -1, "day", "timestamp")
			return result
		}
		return nil
	})

	result.Put("env", func(key string) interface{} {
		return os.Getenv(key)
	})

	for k, v := range UdfRegistry {
		result.Put(k, v)
	}

	return result
}
