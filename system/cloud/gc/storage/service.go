package storage

import (
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/gc"
	"google.golang.org/api/storage/v1"
	"log"
)


const (
	//ServiceID Google StorageService ID.
	ServiceID = "gc/storage"
)


//no operation service
type service struct {
	*endly.AbstractService
}


func (s *service) registerRoutes() {
	client := &storage.Service{}
	routes, err := gc.BuildRoutes(client, nil,  getClient)
	if err != nil {
		log.Printf("unable register service %v actions: %v\n", ServiceID, err)
		return
	}
	for _, route := range routes {
		route.OnRawRequest = InitRequest
		s.Register(route)
	}
}


//New creates a new Storage service
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
