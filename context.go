package endly

import (
	"context"
	"fmt"
	uuid "github.com/satori/go.uuid"
	"github.com/viant/afs/url"
	"github.com/viant/endly/internal/debug"
	"github.com/viant/endly/model/location"
	"github.com/viant/endly/model/msg"
	"github.com/viant/scy/cred/secret"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	tudf "github.com/viant/toolbox/data/udf"
	"github.com/viant/toolbox/storage"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// EndlyPanic env key name to skip recover in case of panic, export ENDLY_PANIC=true
const EndlyPanic = "ENDLY_PANIC"

var serviceManagerKey = (*manager)(nil)
var deferFunctionsKey = (*[]func())(nil)

// Context represents a workflow session context/state
type Context struct {
	context         context.Context
	SessionID       string
	CLIEnabled      bool
	HasLogger       bool
	AsyncUnsafeKeys map[interface{}]bool
	Secrets         *secret.Service
	Wait            *sync.WaitGroup
	Listener        msg.Listener
	Source          *location.Resource
	Debugger        *debug.Debugger

	state   data.Map
	udfs    data.Map
	Logging *bool
	toolbox.Context
	cloned []*Context
	closed int32
}

func (c *Context) Background() context.Context {
	if c.context != nil {
		return c.context
	}
	c.context = context.Background()
	return c.context
}

// Publish publishes event to listeners, it updates current run details like activity workflow name etc ...
func (c *Context) Publish(value interface{}) msg.Event {
	event, ok := value.(msg.Event)
	if !ok {
		event = msg.NewEvent(value)
	}
	event.SetLoggable(c.IsLoggingEnabled())
	if c.Listener != nil {
		c.Listener(event)
	}
	return event
}

// PublishWithStartEvent publishes event to listeners, it updates current run details like activity workflow name etc ...
func (c *Context) PublishWithStartEvent(value interface{}, init msg.Event) msg.Event {
	event := msg.NewEventWithInit(value, init)
	event.SetLoggable(true)
	if c.Listener != nil {
		c.Listener(event)
	}
	return event
}

// SetListener sets context event Listener
func (c *Context) SetListener(listener msg.Listener) {
	c.Listener = listener
}

// IsClosed returns true if it is closed.
func (c *Context) IsClosed() bool {
	return atomic.LoadInt32(&c.closed) == 1
}

// Clone clones the context.
func (c *Context) Clone() *Context {
	if len(c.cloned) == 0 {
		c.cloned = make([]*Context, 0)
	}
	result := &Context{}
	result.Wait = &sync.WaitGroup{}
	result.Context = c.Context.Clone()
	result.state = NewDefaultState(c)
	result.state.Apply(c.state)
	result.SessionID = c.SessionID
	result.Listener = c.Listener
	result.CLIEnabled = c.CLIEnabled
	result.Secrets = c.Secrets
	result.AsyncUnsafeKeys = make(map[interface{}]bool)
	for k, v := range c.AsyncUnsafeKeys {
		result.AsyncUnsafeKeys[k] = v
	}
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

// IsLoggingEnabled returns tru if logging is enabled
func (c *Context) IsLoggingEnabled() bool {
	if c.Logging == nil {
		return true
	}
	return *c.Logging
}

// SetLogging set logging on and off
func (c *Context) SetLogging(flag bool) {
	c.Logging = &flag
}

// ExpandResource substitutes any $ expression with the key value from the state map if it is present.
func (c *Context) ExpandResource(resource *location.Resource) (*location.Resource, error) {
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
			var candidateURL = url.Join(parentCandidate, resource.URL)
			if exists, err := service.Exists(candidateURL); exists && err == nil {
				resource.URL = candidateURL
			}
		}
	}
	var result = location.NewResource(c.Expand(resource.URL), location.WithCredentials(c.Expand(resource.Credentials)))
	result.CustomKey = resource.CustomKey
	return result, nil
}

// Manager returns workflow manager or error
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

// Service returns a service fo provided id or error.
func (c *Context) Service(name string) (Service, error) {
	manager, err := c.Manager()
	if err != nil {
		return nil, err
	}
	return manager.Service(name)
}

// Deffer add function to be executed if context closes. If returns currently registered functions.
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

// State returns a context state map.
func (c *Context) State() data.Map {
	if c.state == nil {
		c.state = NewDefaultState(c)
	}
	return c.state
}

// SetState sets a new state map
func (c *Context) SetState(state data.Map) {
	c.state = state
}

// Expand substitute $ expression if present in the text and state map.
func (c *Context) Expand(text string) string {
	state := c.State()
	return state.ExpandAsText(text)
}

// PublishAndRestore sets supplied value and returns func restoring original values
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

// NewRequest creates a new request for service and action
func (c *Context) NewRequest(serviceName, action string, rawRequest map[string]interface{}) (result interface{}, err error) {
	var service Service
	service, err = c.Service(serviceName)
	if err != nil {
		return nil, err
	}
	route, err := service.Route(action)
	if err != nil {
		return nil, err
	}

	defer func() {
		if toolbox.AsBoolean(os.Getenv("ENDLY_PANIC")) {
			return
		}
		if r := recover(); r != nil {
			var info = toolbox.AsString(rawRequest)
			if JSONSource, err := toolbox.AsJSONText(rawRequest); err == nil {
				info = JSONSource
			}
			err = fmt.Errorf("unable to create %v request: %v, %v", serviceName+":"+action, r, info)
		}
	}()
	request := route.RequestProvider()
	if route.OnRawRequest != nil {
		if err = route.OnRawRequest(c, rawRequest); err != nil {
			return nil, err
		}
	}
	err = toolbox.DefaultConverter.AssignConverted(request, rawRequest)
	return request, err
}

// AsRequest converts a source map into request for provided service and action.
func (c *Context) AsRequest(serviceName, action string, source map[string]interface{}) (request interface{}, err error) {

	expanded := c.state.Expand(source)
	source = toolbox.AsMap(expanded)
	if request, err = c.NewRequest(serviceName, action, source); err != nil {
		return request, fmt.Errorf("unable to create %v request %v", serviceName+":"+action, err)
	}

	return request, err
}

// Close closes this context, it executes all deferred function and set closed flag.
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

var atomicInt int64

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

func NewDefaultState(ctx *Context) data.Map {
	var result = data.NewMap()
	result.Put("ts", time.Now().Unix())
	if ctx.udfs == nil {
		ctx.udfs = predefinedRegistry()
		ctx.udfs.Put("secrets", func(key string) interface{} {
			if ctx.Secrets == nil {
				return ""
			}
			genericCred, err := ctx.Secrets.GetCredentials(ctx.Background(), key)
			if err == nil {
				var result = make(map[string]interface{})
				if err = toolbox.DefaultConverter.AssignConverted(&result, genericCred); err == nil {
					return data.Map(result)
				}
			}
			return ""
		})
	}
	result.Put(data.UDFKey, ctx.udfs)
	return result
}

func predefinedRegistry() data.Map {
	var result = data.NewMap()
	for k, v := range tudf.Predefined {
		result[k] = v
	}
	for k, v := range PredefinedUdfs {
		result.Put(k, v)
	}
	result.Put("minuteofday", func(key string) interface{} {
		now := time.Now()
		if strings.ToLower(key) == "utc" {
			now = now.UTC()
		}
		return int(now.Minute()) + int(now.Hour())*60
	})
	result.Put("weekday", func(key string) interface{} {
		now := time.Now()
		loc := time.UTC
		if key != "" {
			if l, err := time.LoadLocation(key); err != nil {
				loc = l
			}
		}
		return now.In(loc).Weekday()
	})

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

	result.Put("generator", func(key string) interface{} {
		switch key {
		case "next":
			return atomic.AddInt64(&atomicInt, 1)
		case "prev":
			return atomic.AddInt64(&atomicInt, -1)
		case "reset":
			atomic.StoreInt64(&atomicInt, 0)
			return atomic.LoadInt64(&atomicInt)
		default:
			return atomic.LoadInt64(&atomicInt)
		}
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

	//return fraction of elapsed today in supplied key locale, i.e  ${elapsedToday.UTC}
	result.Put("elapsedToday", func(key string) interface{} {
		elapsed, err := toolbox.ElapsedToday(key)
		if err != nil {
			return nil
		}
		return elapsed
	})
	//return fraction of elapsed today in supplied key timezone, i.e  ${remainingToday.Poland}
	result.Put("remainingToday", func(key string) interface{} {
		remainingToday, err := toolbox.RemainingToday(key)
		if err != nil {
			return nil
		}
		return remainingToday
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
	return result
}
