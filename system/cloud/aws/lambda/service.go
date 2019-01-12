package lambda

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/aws"
	"log"
	"sync"
	"sync/atomic"
)

const (
	//ServiceID aws lambda service id.
	ServiceID = "aws/lambda"
)

//no operation service
type service struct {
	*endly.AbstractService
}

func (s *service) recreateFunction(context *endly.Context, request *RecreateFunctionInput) (configuration *lambda.FunctionConfiguration, err error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	wait := &sync.WaitGroup{}
	wait.Add(1)
	var done uint32 = 0

	go func() {
		for ; ; {
			if atomic.LoadUint32(&done) == 1 {
				break
			}
			s.Sleep(context, 2000)
		}
	}()
	go func() {
		defer wait.Done()
		output, _ := client.GetFunction(&lambda.GetFunctionInput{
			FunctionName: request.FunctionName,
		})
		if output != nil && output.Code != nil {
			if _, err = client.DeleteFunction(&lambda.DeleteFunctionInput{
				FunctionName: request.FunctionName,
			}); err != nil {
				return
			}
		}
		lambdaRequest := lambda.CreateFunctionInput(*request)
		configuration, err = client.CreateFunction(&lambdaRequest)
		request.Code = nil
	}()
	wait.Wait()
	atomic.StoreUint32(&done, 1)
	return configuration, err
}

func (s *service) dropFunction(context *endly.Context, request *DropFunctionInput) (interface{}, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	return client.DeleteFunction(&lambda.DeleteFunctionInput{
		FunctionName: request.FunctionName,
	})
}

func (s *service) setupPermission(context *endly.Context, request *SetupPermissionInput) (*lambda.AddPermissionOutput, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	getResponse, _ := client.GetPolicy(&lambda.GetPolicyInput{
		FunctionName: request.FunctionName,
	})
	if getResponse.Policy != nil {
		policy := Policy{}
		if err = json.Unmarshal([]byte(*getResponse.Policy), &policy); err != nil {
			return nil, err
		}
		if len(policy.Statement) > 0 {
			for _, statement := range policy.Statement {
				if statement.Action != *request.Action {
					continue
				}
				if arnLike, ok := statement.Condition["ArnLike"]; ok {
					for _, v := range arnLike {
						if v == *request.SourceArn {
							return &lambda.AddPermissionOutput{Statement:&statement.Sid}, nil
						}
					}
				}
			}
		}
	}
	addPermission := lambda.AddPermissionInput(*request)
	return client.AddPermission(&addPermission)
}

func (s *service) registerRoutes() {
	client := &lambda.Lambda{}
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
		Action:       "recreateFunction",
		RequestInfo:  &endly.ActionInfo{},
		ResponseInfo: &endly.ActionInfo{},
		RequestProvider: func() interface{} {
			return &RecreateFunctionInput{}
		},
		ResponseProvider: func() interface{} {
			return &lambda.FunctionConfiguration{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*RecreateFunctionInput); ok {
				return s.recreateFunction(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action:       "setupPermission",
		RequestInfo:  &endly.ActionInfo{},
		ResponseInfo: &endly.ActionInfo{},
		RequestProvider: func() interface{} {
			return &SetupPermissionInput{}
		},
		ResponseProvider: func() interface{} {
			return &lambda.AddPermissionOutput{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*SetupPermissionInput); ok {
				return s.setupPermission(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

}

//New creates a new AWS Ec2 service.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
