package endly

import (
	"fmt"
	uuid "github.com/satori/go.uuid"
	"github.com/viant/endly/msg"
	"github.com/viant/neatly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/secret"
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

var serviceManagerKey = (*manager)(nil)
var deferFunctionsKey = (*[]func())(nil)

//Context represents a workflow session context/state
type Context struct {
	SessionID       string
	CLIEnabled      bool
	HasLogger       bool
	AsyncUnsafeKeys map[interface{}]bool
	Secrets         *secret.Service
	Wait            *sync.WaitGroup
	Listener        msg.Listener
	Source          *url.Resource
	state           data.Map
	toolbox.Context
	cloned []*Context
	closed int32
}

//Publish publishes event to listeners, it updates current run details like activity workflow name etc ...
func (c *Context) Publish(value interface{}) msg.Event {
	event, ok := value.(msg.Event)
	if !ok {
		event = msg.NewEvent(value)
	}
	if c.Listener != nil {
		c.Listener(event)
	}
	return event
}

//PublishWithStartEvent publishes event to listeners, it updates current run details like activity workflow name etc ...
func (c *Context) PublishWithStartEvent(value interface{}, init msg.Event) msg.Event {
	event := msg.NewEventWithInit(value, init)
	if c.Listener != nil {
		c.Listener(event)
	}
	return event
}

//SetListener sets context event Listener
func (c *Context) SetListener(listener msg.Listener) {
	c.Listener = listener
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
	result.Listener = c.Listener
	result.CLIEnabled = c.CLIEnabled
	result.Secrets = c.Secrets
	result.AsyncUnsafeKeys = make(map[interface{}]bool)
	c.cloned = append(c.cloned, result)
	return result
}

func (c *Context) parentURLCandidates() []string {
	var result = make([]string, 0)
	if c.Source != nil {
		baseURL, _ := toolbox.URLSplit(c.Source.URL)
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
		return nil, msg.ReportError(fmt.Errorf("resource  was empty"))
	}
	if resource.URL == "" {
		return nil, msg.ReportError(fmt.Errorf("url was empty"))
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
	var result = url.NewResource(c.Expand(resource.URL), c.Expand(resource.Credentials))
	if result.ParsedURL == nil {
		return nil, fmt.Errorf("failed to parse URL %v", result.URL)
	}
	result.Cache = c.Expand(resource.Cache)
	result.CacheExpiryMs = resource.CacheExpiryMs
	return result, nil
}

//Manager returns workflow manager or error
func (c *Context) Manager() (Manager, error) {
	if c == nil {
		return nil, fmt.Errorf("context was nil")
	}
	var manager = &manager{}
	if !c.GetInto(serviceManagerKey, &manager) {
		return nil, msg.ReportError(fmt.Errorf("failed to lookup Manager"))
	}
	return manager, nil
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

//Expand substitute $ expression if present in the text and state map.
func (c *Context) Expand(text string) string {
	state := c.State()
	return state.ExpandAsText(text)
}

//PublishAndRestore sets supplied value and returns func restoring original values
func (s *Context) PublishAndRestore(values map[string]interface{}) func() {
	var backup = map[string]interface{}{}
	for k, v := range values {
		if value, has := s.state.GetValue(k); has {
			backup[k] = value
		}
		s.state.SetValue(k, v)
	}
	return func() {
		for k, v := range backup {
			s.state.SetValue(k, v)
		}
	}
}

//NewRequest creates a new request for service and action
func (c *Context) NewRequest(serviceName, action string) (interface{}, error) {
	service, err := c.Service(serviceName)
	if err != nil {
		return nil, err
	}
	route, err := service.Route(action)
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
			err = fmt.Errorf("failed to create request, unable to cast %v into %T, %v", source, request, r)
		}
	}()
	expanded := c.state.Expand(source)
	source = toolbox.AsMap(expanded)
	err = toolbox.DefaultConverter.AssignConverted(request, source)
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

func (c *Context) MakeAsyncSafe() *msg.Events {
	for k := range c.AsyncUnsafeKeys {
		c.Context.Remove(k)
	}
	result := msg.NewEvents()
	c.Listener = result.AsListener()
	return result
}

var yyyyMMDDLayout = toolbox.DateFormatToLayout("yyyy-MM-dd")
var yyyMMDDHHMMSSLayout = toolbox.DateFormatToLayout("yyyy-MM-dd hh:mm:ss")
var numberDateLayout = toolbox.DateFormatToLayout("yyyyMMddhhmmSSS")

/*
NewDefaultState returns a new default state.
It comes with the following registered keys:
	* rand - random int64
	* date -  current date formatted as yyyy-MM-dd
	* time - current time formatted as yyyy-MM-dd hh:mm:ss
	* ts - current timestamp formatted  as yyyyMMddhhmmSSS
	* timestamp.XXX - timestamp in ms where XXX is time diff expression i.e 3DaysAgo, tomorrow, hourAhead
	* unix.XXX - timestamp in sec where XXX is time diff expression i.e 3DaysAgo, tomorrow, hourAhead
	* tzTime.XXX - RFC3339 formatted time where XXX is time diff expression i.e 3DaysAgo, tomorrow, hourAhead
	* tmpDir - temp directory
	* uuid.next - generate unique id
	* uuid.Get - returns previously generated unique id, or generate new
	*.env.XXX where XXX is the ID of the env variable to return
	* all UFD registry functions
*/
func NewDefaultState() data.Map {
	var result = data.NewMap()
	var now = time.Now()
	source := rand.NewSource(now.UnixNano())
	result.Put("rand", source.Int63())
	result.Put("date", now.Format(yyyyMMDDLayout))
	result.Put("time", now.Format(yyyMMDDHHMMSSLayout))
	result.Put("ts", now.Unix())

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

	//returns time in ms
	result.Put("timestamp", func(key string) interface{} {
		timeAt, err := toolbox.TimeAt(key)
		if err != nil {
			return nil
		}
		return int(timeAt.Unix()+timeAt.UnixNano()) / 1000000
	})
	//return time in sec
	result.Put("unix", func(key string) interface{} {
		timeAt, err := toolbox.TimeAt(key)
		if err != nil {
			return nil
		}
		return int(timeAt.Unix()+timeAt.UnixNano()) / 1000000000
	})

	//return formatted time with time.RFC3339 yyyy-MM-ddThh:mm:ss.SSS Z  i.e ${tzTime.4daysAgoInUTC}

	result.Put("tzTime", func(key string) interface{} {
		timeAt, err := toolbox.TimeAt(key)
		if err != nil {
			return nil
		}
		return timeAt.Format(time.RFC3339)
	})

	result.Put("env", func(key string) interface{} {
		return os.Getenv(key)
	})

	neatly.AddStandardUdf(result)
	for k, v := range UdfRegistry {
		result.Put(k, v)
	}
	return result
}
