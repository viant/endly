package endly

import (
	"fmt"
	uuid "github.com/satori/go.uuid"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/storage"
	"github.com/viant/toolbox/url"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var converter = toolbox.NewColumnConverter("yyyy-MM-dd HH:ss")

var serviceManagerKey = (*manager)(nil)
var deferFunctionsKey = (*[]func())(nil)

//WorkflowKey context.State workflow key
var WorkflowKey = (*Workflow)(nil)

//Context represents a workflow session context/state
type Context struct {
	SessionID  string
	CLIEnabled bool
	Wait       *sync.WaitGroup
	listener   EventListener
	Workflows  *Workflows
	state      data.Map
	toolbox.Context
	cloned []*Context
	closed int32
}

//Publish publishes event to listeners, it updates current run details like activity workflow name etc ...
func (c *Context) Publish(value interface{}) *Event {
	var workflow = c.Workflows.Last()
	var workflowName = ""
	if workflow != nil {
		workflowName = workflow.Name
	}
	state := c.state
	var activity = &Activity{
		Workflow: workflowName,
	}
	if state.Has(TaskKey) {
		task := state.GetString(TaskKey)
		activity.Task = task
	}
	if state.Has(ActivityKey) {
		activity, _ = state.Get(ActivityKey).(*Activity)
	}
	var event = &Event{
		Timestamp: time.Now(),
		Activity:  activity,
		Value:     value,
	}
	if c.listener != nil {
		c.listener(event)
	}
	return event
}

//SetListener sets context event listener
func (c *Context) SetListener(listener EventListener) {
	c.listener = listener
}

//IsClosed returns true if it is closed.
func (c *Context) IsClosed() bool {
	return atomic.LoadInt32(&c.closed) == 1
}

//Clone clones the context.
func (c *Context) Clone() *Context {
	if len(c.cloned) == 0 {
		c.cloned = make([]*Context, 0)
	}
	result := &Context{}
	result.Wait = &sync.WaitGroup{}
	result.Context = c.Context.Clone()
	result.state = NewDefaultState()
	result.state.Apply(c.state)
	result.SessionID = c.SessionID
	result.Workflows = c.Workflows
	result.listener = c.listener
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

//ExpandResource substitutes any $ expression with the key value from the state map if it is present.
func (c *Context) ExpandResource(resource *url.Resource) (*url.Resource, error) {
	if resource == nil {
		return nil, reportError(fmt.Errorf("resource  was empty"))
	}
	if resource.URL == "" {
		return nil, reportError(fmt.Errorf("url was empty"))
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
		return nil, fmt.Errorf("failed to parse URL %v", result.URL)
	}
	result.Cache = c.Expand(resource.Cache)
	result.CacheExpiryMs = resource.CacheExpiryMs
	return result, nil
}

//Manager returns workflow manager or error
func (c *Context) Manager() (Manager, error) {
	var manager = &manager{}
	if !c.GetInto(serviceManagerKey, &manager) {
		return nil, reportError(fmt.Errorf("failed to lookup Service"))
	}
	return manager, nil
}

//TerminalSessions returns client sessions
func (c *Context) TerminalSessions() SystemTerminalSessions {
	var result *SystemTerminalSessions
	if !c.Contains(systemTerminalSessionsKey) {
		var sessions SystemTerminalSessions = make(map[string]*SystemTerminalSession)
		result = &sessions
		_ = c.Put(systemTerminalSessionsKey, result)
	} else {
		c.GetInto(systemTerminalSessionsKey, &result)
	}
	return *result
}

//Service returns a service fo provided id or error.
func (c *Context) Service(name string) (Service, error) {
	manager, err := c.Manager()
	if err != nil {
		return nil, err
	}
	return manager.Service(name)
}

//Deffer add function to be executed if context closes. If returns currently registered functions.
func (c *Context) Deffer(functions ...func()) []func() {
	var result *[]func()
	if !c.Contains(deferFunctionsKey) {
		var functions = make([]func(), 0)
		result = &functions
		_ = c.Put(deferFunctionsKey, result)
	} else {
		c.GetInto(deferFunctionsKey, &result)
	}

	*result = append(*result, functions...)
	_ = c.Put(deferFunctionsKey, &result)
	return *result
}

//State returns a context state map.
func (c *Context) State() data.Map {
	if c.state == nil {
		c.state = NewDefaultState()
	}
	return c.state
}

//SetState sets a new state map
func (c *Context) SetState(state data.Map) {
	c.state = state
}

//Workflow returns the master workflow
func (c *Context) Workflow() *Workflow {
	var result *Workflow
	if !c.Contains(WorkflowKey) {
		return nil
	}
	c.GetInto(WorkflowKey, &result)
	return result
}

//OperatingSystem returns operating system for provide session
func (c *Context) OperatingSystem(sessionName string) *OperatingSystem {
	var sessions = c.TerminalSessions()

	if session, has := sessions[sessionName]; has {
		return session.OperatingSystem
	}
	return nil
}

//Expand substitute $ expression if present in the text and state map.
func (c *Context) Expand(text string) string {
	state := c.State()
	return state.ExpandAsText(text)
}

//NewRequest creates a new request for service and action
func (c *Context) NewRequest(serviceName, action string) (interface{}, error) {
	service, err := c.Service(serviceName)
	if err != nil {
		return nil, err
	}
	route, err := service.ServiceActionRoute(action)
	if err != nil {
		return nil, err
	}
	return route.RequestProvider(), nil
}

//AsRequest converts a source map into request for provided service and action.
func (c *Context) AsRequest(serviceName, action string, source map[string]interface{}) (request interface{}, err error) {
	if request, err = c.NewRequest(serviceName, action); err != nil {
		return request, err
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("failed to create request, unable to case %v into %T, %v", source, request, r)
		}
	}()
	err = converter.AssignConverted(request, source)
	return request, err
}

//Close closes this context, it executes all deferred function and set closed flag.
func (c *Context) Close() {
	atomic.StoreInt32(&c.closed, 1)
	for _, context := range c.cloned {
		context.Close()
	}
	for _, function := range c.Deffer() {
		function()
	}
}

//MakeAsyncSafe makes this contex async safe
func (c *Context) MakeAsyncSafe() *Events {
	c.Context.Remove(systemTerminalSessionsKey)
	result := &Events{
		mutex:  &sync.Mutex{},
		Events: make([]*Event, 0),
	}
	c.listener = result.AsEventListener()
	return result
}

/*
NewDefaultState returns a new default state.
It comes with the following registered keys:
	* rand - random int64
	*  date -  current date formatted as yyyy-MM-dd
	* time - current time formatted as yyyy-MM-dd hh:mm:ss
	* ts - current timestamp formatted  as yyyyMMddhhmmSSS
	* timestamp.yesterday - timestamp in ms
	* timestamp.now - timestamp in ms
	* timestamp.tomorrow - timestamp in ms
	* tmpDir - temp directory
	* uuid.next - generate unique id
	* uuid.Get - returns previously generated unique id, or generate new
	*.end.XXX where XXX is the ID of the env variable to return
	* all UFD registry functions
*/
func NewDefaultState() data.Map {
	var result = data.NewMap()
	var now = time.Now()
	source := rand.NewSource(now.UnixNano())
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
			var err error
			cachedUUID, err = uuid.NewV4()
			if err != nil {
				return ""
			}
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
