package compute

import (
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/gcp"
	"google.golang.org/api/compute/v1"
	"log"
)

const (
	//ServiceID Google Cloud Compute Service ID.
	ServiceID = "gcp/compute"
)

// no operation service
type service struct {
	*endly.AbstractService
}

func (s *service) registerRoutes() {
	client := &compute.Service{}
	routes, err := gcp.BuildRoutes(client, nil, getClient)
	if err != nil {
		log.Printf("unable register service %v actions: %v\n", ServiceID, err)
		return
	}
	for _, route := range routes {
		route.OnRawRequest = InitRequest
		s.Register(route)
	}
}

// New creates a new Compute service.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
