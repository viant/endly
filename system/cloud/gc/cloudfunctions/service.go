package cloudfunctions

import (
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/gc"
	"google.golang.org/api/cloudfunctions/v1"
	"log"
)



const (
	//ServiceID Google Cloud Function Service Id
	ServiceID = "gc/cloudfunctions"
)


//no operation service
type service struct {
	*endly.AbstractService
}


func (s *service) registerRoutes() {
	client := &cloudfunctions.Service{}
	routes, err := gc.BuildRoutes(client, getClient)
	if err != nil {
		log.Printf("unable register service %v actions: %v\n", ServiceID, err)
		return
	}
	for _, route := range routes {
		route.OnRawRequest = InitRequest
		s.Register(route)
	}
}


//New creates a new Dataflow service
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
