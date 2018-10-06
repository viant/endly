package endly

import (
	"github.com/pkg/errors"
	"github.com/viant/toolbox/data"
)

//UdfRegistry represents a udf registry
var UdfRegistry = make(map[string]func(source interface{}, state data.Map) (interface{}, error))


//UdfRegistryProvider represents udf registry provider (i.e. to register parameterized udf dynamically)
var UdfRegistryProvider = make(map[string]func(args ... interface{}) (func(source interface{}, state data.Map) (interface{}, error), error))

type UdfProvider struct {
	Id string
	Provider string
	Params []interface{}
}

//ServiceProvider represents a service provider
type ServiceProvider func() Service

//ServiceRegistry  represents a service registry
type ServiceRegistry []ServiceProvider

//Register register service provider.
func (r *ServiceRegistry) Register(serviceProvider ServiceProvider) error {
	if serviceProvider == nil {
		return errors.New("provider was empty")
	}
	*r = append(*r, serviceProvider)
	return nil
}

var registry ServiceRegistry = make([]ServiceProvider, 0)

//Registry global service provider registry
var Registry = &registry


