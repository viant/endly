package cloudwatch

import (
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/viant/endly"
	"github.com/viant/endly/service/system/cloud/aws"
	"log"
)

const (
	//ServiceID aws Cloudwatch service id.
	ServiceID = "aws/cloudwatch"
)

// no operation service
type service struct {
	*endly.AbstractService
}

func (s *service) registerRoutes() {
	client := &cloudwatch.CloudWatch{}
	routes, err := aws.BuildRoutes(client, getClient)
	if err != nil {
		log.Printf("unable register service %v actions: %v\n", ServiceID, err)
		return
	}
	for _, route := range routes {
		route.OnRawRequest = setClient
		s.Register(route)
	}
}

// New creates a new AWS Cloudwatch service.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
