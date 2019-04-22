package rds

import (
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/aws"
	"log"
)

const (
	//ServiceID Amazon Relational Database Service service id.
	ServiceID = "aws/rds"
)

//no operation service
type service struct {
	*endly.AbstractService
}

func (s *service) registerRoutes() {
	client := &rds.RDS{}
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

//New creates a new AWS DynamoDB service.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
