package endly

import (
	"github.com/viant/toolbox"
)

//ServiceActionMeta represents service action meta
type ServiceActionMeta struct {
	*ServiceActionRoute
	Request      interface{}
	RequestMeta  *toolbox.StructMeta
	Response     interface{}
	ResponseMeta *toolbox.StructMeta
}

//MetaService represents service action meta service
type MetaService struct {
	Manager
}

//Lookup returns service action info for supplied serviceID and action
func (m *MetaService) Lookup(serviceID, action string) (*ServiceActionMeta, error) {
	var result = &ServiceActionMeta{}
	context := m.NewContext(toolbox.NewContext())
	service, err := context.Service(serviceID)
	if err != nil {
		return nil, err
	}
	result.ServiceActionRoute, err = service.ServiceActionRoute(action)
	if err != nil {
		return nil, err
	}
	request := result.RequestProvider()
	toolbox.InitStruct(request)
	result.Request = request
	result.RequestMeta = toolbox.GetStructMeta(request)

	response := result.ResponseProvider()
	toolbox.InitStruct(response)
	result.Response = response
	result.ResponseMeta = toolbox.GetStructMeta(response)
	return result, nil
}

//NewMetaService creates a new meta service
func NewMetaService() *MetaService {
	return &MetaService{NewManager()}
}
