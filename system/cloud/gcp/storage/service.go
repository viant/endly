package storage

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/gcp"
	"google.golang.org/api/storage/v1"
	"log"
)

const (
	//ServiceID Google StorageService ID.
	ServiceID = "gcp/storage"
)

//no operation service
type service struct {
	*endly.AbstractService
}

func (s *service) registerRoutes() {
	client := &storage.Service{}
	routes, err := gcp.BuildRoutes(client, nil, getClient)
	if err != nil {
		log.Printf("unable register service %v actions: %v\n", ServiceID, err)
		return
	}

	for _, route := range routes {
		route.OnRawRequest = InitRequest
		s.Register(route)
	}

	s.Register(&endly.Route{
		Action: "setNotification",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "setNotification", &SetupNotificationRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &SetupNotificationResponse{}),
		},
		RequestProvider: func() interface{} {
			return &SetupNotificationRequest{}
		},
		ResponseProvider: func() interface{} {
			return &SetupNotificationResponse{}
		},
		OnRawRequest: InitRequest,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*SetupNotificationRequest); ok {
				output, err := s.SetNotification(context, req)
				if err != nil {
					return nil, err
				}
				if context.IsLoggingEnabled() {
					context.Publish(gcp.NewOutputEvent("...", "setNotification", output))
				}
				return output, err
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//New creates a new service
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
