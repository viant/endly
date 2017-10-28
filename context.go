package endly

import (
	"fmt"
	"github.com/pkg/errors"
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
	"sync/atomic"
	"time"
)

var converter = toolbox.NewColumnConverter("yyyy-MM-dd HH:ss")

var serviceManagerKey = (*manager)(nil)
var deferFunctionsKey = (*[]func())(nil)
var workflowKey = (*Workflow)(nil)

//Context represents a workflow session context/state
type Context struct {
	SessionID string
	state     data.Map
	toolbox.Context
	Events      *Events
	EventLogger *EventLogger
	Workflows   *Workflows
	cloned      []*Context
	closed      int32
}

//IsClosed returns true if it is closed.
func (c *Context) IsClosed() bool {
	return atomic.LoadInt32(&c.closed) == 1
}

func reportError(err error) error {
	fileName, funcName, line := toolbox.CallerInfo(4)
	return fmt.Errorf("%v at %v:%v -> %v", err, fileName, line, funcName)
}

//Clone clones the context.
func (c *Context) Clone() *Context {
	if len(c.cloned) == 0 {
		c.cloned = make([]*Context, 0)
	}
	result := &Context{}
	result.Context = c.Context.Clone()
	result.Events = c.Events
	result.state = NewDefaultState()
	result.state.Apply(c.state)
	result.SessionID = c.SessionID
	result.Workflows = c.Workflows
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

//ExpandResource substitutes any $ expression with the key value from the state map if it is present.
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

//Manager returns workflow manager or error
func (c *Context) Manager() (Manager, error) {
	var manager = &manager{}
	if !c.GetInto(serviceManagerKey, &manager) {
		return nil, reportError(fmt.Errorf("Failed to lookup Manager"))
	}
	return manager, nil
}

//TerminalSessions returns client sessions
func (c *Context) TerminalSessions() SystemTerminalSessions {
	var result *SystemTerminalSessions
	if !c.Contains(clientSessionKey) {
		var sessions SystemTerminalSessions = make(map[string]*SystemTerminalSession)
		result = &sessions
		c.Put(clientSessionKey, result)
	} else {
		c.GetInto(clientSessionKey, &result)
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
		c.Put(deferFunctionsKey, result)
	} else {
		c.GetInto(deferFunctionsKey, &result)
	}

	*result = append(*result, functions...)
	c.Put(deferFunctionsKey, &result)
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
	if !c.Contains(workflowKey) {
		return nil
	}
	c.GetInto(workflowKey, &result)
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

//ExecuteAsSuperUser executes provided command as super user.
func (c *Context) ExecuteAsSuperUser(target *url.Resource, command *ManagedCommand) (*CommandResponse, error) {
	superUserRequest := superUserCommandRequest{
		Target:        target,
		MangedCommand: command,
	}
	request, err := superUserRequest.AsCommandRequest(c)
	if err != nil {
		return nil, err
	}
	return c.Execute(target, request.ManagedCommand)
}

//Execute execute shell command
func (c *Context) Execute(target *url.Resource, command interface{}) (*CommandResponse, error) {
	if command == nil {
		return nil, nil
	}
	var commandRequest *ManagedCommandRequest
	switch actualCommand := command.(type) {
	case *CommandRequest:
		actualCommand.Target = target
		commandRequest = actualCommand.AsManagedCommandRequest()
	case *ManagedCommand:
		commandRequest = NewManagedCommandRequest(target, actualCommand)
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
	execService, err := c.Service(SystemExecServiceID)
	if err != nil {
		return nil, err
	}
	response := execService.Run(c, commandRequest)
	if response.Error != "" {
		return nil, errors.New(response.Error)
	}
	if commandResult, ok := response.Response.(*CommandResponse); ok {
		return commandResult, nil
	}
	return nil, nil
}

//Copy transfer source into target url, it takes also exand flag to indicate variable substitution.
func (c *Context) Copy(expand bool, source, target *url.Resource) (interface{}, error) {
	return c.Transfer([]*Transfer{{
		Source: source,
		Target: target,
		Expand: expand}}...)
}

//Transfer transfer data for provided transfer definition.
func (c *Context) Transfer(transfers ...*Transfer) (interface{}, error) {
	if transfers == nil {
		return nil, nil
	}
	transferService, err := c.Service(TransferServiceID)
	if err != nil {
		return nil, err
	}
	response := transferService.Run(c, &TransferCopyRequest{Transfers: transfers})
	if response.Error != "" {
		return nil, errors.New(response.Error)
	}
	return nil, nil
}

//Expand substitute $ expression if present in the text and state map.
func (c *Context) Expand(text string) string {
	state := c.State()
	return state.ExpandAsText(text)
}

//AsRequest converts a source map into request for provided service and action.
func (c *Context) AsRequest(serviceName, action string, source map[string]interface{}) (interface{}, error) {
	service, err := c.Service(serviceName)
	if err != nil {
		return nil, err
	}
	request, err := service.NewRequest(action)
	if err != nil {
		return nil, err
	}
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
	* uuid.get - returns previously generated unique id, or generate new
	*.end.XXX where XXX is the Id of the env variable to return
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
