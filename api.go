package endly

import (
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"sync"
)

//AppName represents endly application name
const AppName = "endly"

//Namespace represents endly namespace
const Namespace = "github.com/viant/endly/"

//Manager represents a endly service manager
type Manager interface {
	//Name returns an application ID
	Name() string

	//Version returns an application version
	Version() string

	//Service return a workflow service for provided ID, request,  or error
	Service(input interface{}) (Service, error)

	//Register register service in this manager
	Register(service Service)

	//NewContext returns new workflow context.
	NewContext(context toolbox.Context) *Context

	//Run run requests
	Run(context *Context, request interface{}) (interface{}, error)
}

//Service represents a service
type Service interface {
	//service id
	ID() string

	//service state map
	State() data.Map

	//Run service action for supported request types.
	Run(context *Context, request interface{}) *ServiceResponse

	//Route returns service action route
	Route(action string) (*Route, error)

	Mutex() *sync.RWMutex

	Actions() []string
}

//Validator represents generic validator
type Validator interface {
	Validate() error
}

//Initializer represents generic initializer
type Initializer interface {
	Init() error
}

//ServiceResponse service response
type ServiceResponse struct {
	Status   string
	Error    string
	Response interface{}
	Err      error
}

//Route represents service action route
type Route struct {
	Action           string
	RequestInfo      *ActionInfo
	ResponseInfo     *ActionInfo
	RequestProvider  func() interface{}
	ResponseProvider func() interface{}
	Handler          func(context *Context, request interface{}) (interface{}, error)
}

//Description represents example use case
type UseCase struct {
	Description string
	Data        string
}

//ActionInfo represent an action info
type ActionInfo struct {
	Description string
	Examples    []*UseCase
}
