package networking

import (
	"github.com/viant/endly"
	"github.com/viant/endly/system/kubernetes/shared"
	"k8s.io/client-go/kubernetes/fake"
	"log"
)

const (
	//ServiceID Kubernetes networking service ID.
	ServiceID = "kubernetes/networking"
)

// no operation service
type service struct {
	*endly.AbstractService
}

func (s *service) registerRoutes() {
	clientSet := fake.NewSimpleClientset()

	s.registerClientRoutes(clientSet.NetworkingV1(), "Networking")
}

func (s *service) registerClientRoutes(client interface{}, clientPrefix string) {
	routes, err := shared.BuildRoutes(client, clientPrefix)
	if err != nil {
		log.Printf("unable register service %v actions: %v\n", ServiceID, err)
		return
	}
	for _, route := range routes {
		s.Register(route)
	}

}

// New creates a new service
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
