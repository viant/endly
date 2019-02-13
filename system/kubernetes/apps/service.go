package apps

import (
	"github.com/viant/endly"
	"github.com/viant/endly/system/kubernetes/shared"
	"k8s.io/client-go/kubernetes/fake"
	"log"
)

const (
	//ServiceID Kubernetes Apps service ID.
	ServiceID = "kubernetes/apps"
)

//no operation service
type service struct {
	*endly.AbstractService
}

func (s *service) registerRoutes() {
	clientSet := fake.NewSimpleClientset()
	s.registerClientRoutes(clientSet.AppsV1(), "Apps", "")
	s.registerClientRoutes(clientSet.AppsV1beta2(), "Apps", "v1b2")
}

func (s *service) registerClientRoutes(client interface{}, clientPrefix, actionPrefix string) {
	routes, err := shared.BuildRoutesWithPrefix(client, clientPrefix, actionPrefix)
	if err != nil {
		log.Printf("unable register service %v actions: %v\n", ServiceID, err)
		return
	}
	for _, route := range routes {
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
