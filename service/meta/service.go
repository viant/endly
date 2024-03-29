package meta

import (
	"encoding/json"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
)

// Service represents service action meta service
type Service struct {
	endly.Manager
}

// Lookup returns service action info for supplied serviceID and action
func (m *Service) Lookup(serviceID, action string) (*Action, error) {
	var result = &Action{}
	context := m.NewContext(toolbox.NewContext())
	service, err := context.Service(serviceID)
	if err != nil {
		return nil, err
	}
	result.Route, err = service.Route(action)
	if err != nil {
		return nil, err
	}
	request := result.RequestProvider()
	toolbox.InitStruct(request)
	result.Request = request
	result.RequestMeta = toolbox.GetStructMeta(request)
	JSON, err := json.Marshal(request)
	if err == nil {
		if isEmpty := toolbox.AsString(JSON) == "{}"; isEmpty {
			result.Request = result.RequestMeta.Message()
		}
	}
	response := result.ResponseProvider()
	toolbox.InitStruct(response)
	result.Response = response
	result.ResponseMeta = toolbox.GetStructMeta(response)
	return result, nil
}

// New creates a new meta service
func New() *Service {
	return &Service{endly.New()}
}
