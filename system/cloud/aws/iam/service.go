package iam

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"log"
)


const (
	//ServiceID aws iam service id.
	ServiceID = "aws/iam"
)


//no operation service
type service struct {
	*endly.AbstractService
}


func (s *service) dropRole(context *endly.Context, request *DropRoleInput) (interface{}, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	policies, err := client.ListRolePolicies(&iam.ListRolePoliciesInput{
		RoleName:request.RoleName,
	})
	if err != nil {
		return nil, err
	}
	if policies != nil && len(policies.PolicyNames) > 0 {
		for _, policy := range policies.PolicyNames {
			if _, err = client.DeleteRolePolicy(&iam.DeleteRolePolicyInput{
				PolicyName: policy,
				RoleName:   request.RoleName,
			});err!= nil {
				return nil, fmt.Errorf("failed to delete policy %v",err)
			}
		}
	}
	output, err := client.ListAttachedRolePolicies(&iam.ListAttachedRolePoliciesInput{
		RoleName:request.RoleName,
	})
	if err != nil {
		return nil, err
	}
	if output != nil && len(output.AttachedPolicies) > 0 {
		for _, attached := range output.AttachedPolicies {
			if _, err = client.DetachRolePolicy(&iam.DetachRolePolicyInput{
				PolicyArn:attached.PolicyArn,
				RoleName:request.RoleName,
			});err!= nil {
				return nil, fmt.Errorf("failed to detach policy %v",err)
			}
		}
	}
	return client.DeleteRole(&iam.DeleteRoleInput{
		RoleName:request.RoleName,
	})
}


func (s *service) recreateRole(context *endly.Context, request *RecreateRoleInput) (interface{}, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	output, err := client.GetRole(&iam.GetRoleInput{
		RoleName:request.RoleName,
	})
	if err == nil && output.Role != nil {
		if _, err = s.dropRole(context, &DropRoleInput{
			RoleName:request.RoleName,
		});err != nil {
			return nil, err
		}
	}
	iamRequest:= iam.CreateRoleInput(*request)
	return client.CreateRole(&iamRequest)
}

func (s *service) registerRoutes() {
	client := &iam.IAM{}
	routes, err := aws.BuildRoutes(client, getClient)
	if err != nil {
		log.Printf("unable register service %v actions: %v\n", ServiceID, err)
		return
	}
	for _, route := range routes {
		route.OnRawRequest = setClient
		s.Register(route)
	}

	s.Register(&endly.Route{
		Action:"recreateRole",
		RequestInfo:&endly.ActionInfo{
			Description: "recreate role",
		},
		ResponseInfo: &endly.ActionInfo{
			Description: "response",
		},
		RequestProvider: func() interface{} {
			return &RecreateRoleInput{}
		},
		ResponseProvider: func() interface{} {
			return &iam.CreateRoleOutput{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*RecreateRoleInput); ok {
				return s.recreateRole(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}





//New creates a new AWS IAM service.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
