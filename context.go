package endly

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/viant/endly/common"
	"github.com/viant/toolbox"
	"math/rand"
	"net/url"
	"os"
	"os/exec"
	uuid "github.com/satori/go.uuid"
	"path"
	"strings"
	"time"
)

//TODO Execution detail Tracking of all run (time taken, request, response)

var converter = toolbox.NewColumnConverter("yyyy-MM-dd HH:ss")

type StateKey *common.Map

var serviceManagerKey = (*manager)(nil)
var deferFunctionsKey = (*[]func())(nil)
var workflowKey = (*Workflow)(nil)

type Context struct {
	SessionId     string
	state         common.Map
	toolbox.Context
	Events        *Events
	workflowStack *[]*Workflow
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
	result := &Context{}
	result.Context = c.Context.Clone()
	result.Events = c.Events
	result.state = NewDefaultState()
	result.state.Apply(c.state)
	result.SessionId = c.SessionId
	result.workflowStack = c.workflowStack
	return result
}

func (c *Context) ExpandResource(resource *Resource) (*Resource, error) {
	var err error
	if resource == nil {
		return nil, reportError(fmt.Errorf("Resource was empty"))
	}
	if resource.URL == "" {
		return nil, reportError(fmt.Errorf("URL was empty"))
	}

	if !strings.Contains(resource.URL, "://") {
		if workflow := c.Workflow(); workflow != nil && workflow.source != nil {
			baseURL, _ := toolbox.URLSplit(workflow.source.URL)
			resource.URL = toolbox.URLPathJoin(baseURL, resource.URL)
		}
	}

	var result = &Resource{
		URL:         c.Expand(resource.URL),
		Credential:  c.Expand(resource.Credential),
		Name:        c.Expand(resource.Name),
		Version:     resource.Version,
		Type:        c.Expand(resource.Type),
		Cache:       c.Expand(resource.Cache),
		CacheExpiry: resource.CacheExpiry,
	}
	result.ParsedURL, err = url.Parse(result.URL)
	if err != nil {
		return nil, reportError(fmt.Errorf("Failed to parse URL: %v %v", result.URL, err))
	}
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

func (c *Context) State() common.Map {
	if c.state == nil {
		c.state = NewDefaultState()
	}
	return c.state
}

func (c *Context) SetState(state common.Map) {
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

func (c *Context) ExecuteAsSuperUser(target *Resource, command *ManagedCommand) (*CommandInfo, error) {
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

func (c *Context) Execute(target *Resource, command interface{}) (*CommandInfo, error) {
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

func (c *Context) Copy(expand bool, source, target *Resource) (interface{}, error) {
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
	return ExpandAsText(state, text)
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
	for _, function := range c.Deffer() {
		function()
	}
}

func NewDefaultState() common.Map {
	var result = common.NewMap()
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
			cachedUUID = uuid.NewV1()
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
	return result
}
