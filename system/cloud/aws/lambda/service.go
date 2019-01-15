package lambda

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/aws"
	"github.com/viant/endly/system/cloud/aws/iam"
	"github.com/viant/toolbox"
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
		configuration, err = s.recreateFunctionInBackground(context, request)
	}()
	wait.Wait()
	atomic.StoreUint32(&done, 1)
	return configuration, err
}

func (s *service) recreateFunctionInBackground(context *endly.Context, request *RecreateFunctionInput) (configuration *lambda.FunctionConfiguration, err error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
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

//*iam.Statement
func (s *service) setupPermission(context *endly.Context, request *SetupPermissionInput) (interface{}, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	getResponse, _ := client.GetPolicy(&lambda.GetPolicyInput{
		FunctionName: request.FunctionName,
	})
	if getResponse.Policy != nil {
		policy := &iam.PolicyDocument{}
		if err = json.Unmarshal([]byte(*getResponse.Policy), &policy); err != nil {
			return nil, err
		}
		if len(policy.Statement) > 0 {
			for _, statement := range policy.Statement {
				action, _ :=statement.Action.Value()
				if toolbox.AsString(action) != *request.Action {
					continue
				}
				condition, _ := statement.Condition.Value()
				if conditionMap, ok := condition.(map[string]interface{}); ok {
					if arnLike, ok := conditionMap["ArnLike"]; ok {
						if arnLikeMap, ok := arnLike.(map[string]interface{}); ok {
							for _, v := range arnLikeMap {
								if v == *request.SourceArn {
									return &lambda.AddPermissionOutput{Statement: statement.Sid}, nil
								}
							}
						}
					}
				}
			}
		}
	}
	addPermission := lambda.AddPermissionInput(*request)
	return client.AddPermission(&addPermission)
}

func (s *service) setupFunction(context *endly.Context, request *SetupFunctionInput) (output *SetupFunctionOutput, err error) {
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
		output, err = s.setupFunctionInBackground(context, request)
	}()
	wait.Wait()
	atomic.StoreUint32(&done, 1)
	return output, err
}

func (s *service) setupFunctionInBackground(context *endly.Context, request *SetupFunctionInput) (*SetupFunctionOutput, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	output := &SetupFunctionOutput{}
	output.RoleInfo = &iam.GetRoleInfoOutput{}
	if err = endly.Run(context, &request.SetupRolePolicyInput, &output.RoleInfo); err != nil {
		return nil, err
	}
	request.Role = output.RoleInfo.Role.Arn
	var functionConfig *lambda.FunctionConfiguration
	functionOutput, foundErr := client.GetFunction(&lambda.GetFunctionInput{
		FunctionName: request.FunctionName,
	});
	if foundErr == nil {
		functionConfig = functionOutput.Configuration
		createFunction := request.CreateFunctionInput
		if _, err = client.UpdateFunctionConfiguration(&lambda.UpdateFunctionConfigurationInput{
			FunctionName:     request.FunctionName,
			Description:      request.Description,
			Handler:          request.Handler,
			KMSKeyArn:        request.KMSKeyArn,
			MemorySize:       request.MemorySize,
			Role:             request.Role,
			Runtime:          request.Runtime,
			Timeout:          request.Timeout,
			TracingConfig:    request.TracingConfig,
			VpcConfig:        request.VpcConfig,
			Environment:      request.Environment,
			DeadLetterConfig: request.DeadLetterConfig,
		}); err != nil {
			return nil, err
		}

		if createFunction.Code.ZipFile != nil && ! hasDataChanged(createFunction.Code.ZipFile, *functionConfig.CodeSha256) {
			output.FunctionConfiguration = functionOutput.Configuration
			return output, nil
		}
		if functionConfig, err = client.UpdateFunctionCode(&lambda.UpdateFunctionCodeInput{
			FunctionName:    createFunction.FunctionName,
			ZipFile:         createFunction.Code.ZipFile,
			S3Bucket:        createFunction.Code.S3Bucket,
			S3Key:           createFunction.Code.S3Key,
			S3ObjectVersion: createFunction.Code.S3ObjectVersion,
			Publish:         createFunction.Publish,
		}); err != nil {
			return nil, err
		}

	} else {
		if functionConfig, err = client.CreateFunction(&request.CreateFunctionInput); err != nil {
			return nil, err
		}
	}
	output.FunctionConfiguration = functionConfig
	return output, nil
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
		Action: "recreateFunction",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "recreateFunction", &RecreateFunctionInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &lambda.FunctionConfiguration{}),
		},
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
		Action: "setupPermission",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "setupPermission", &SetupPermissionInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &lambda.AddPermissionOutput{}),
		},
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
	s.Register(&endly.Route{
		Action: "setupFunction",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "setupFunction", &SetupFunctionInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &SetupFunctionOutput{}),
		},
		RequestProvider: func() interface{} {
			return &SetupFunctionInput{}
		},
		ResponseProvider: func() interface{} {
			return &SetupFunctionOutput{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*SetupFunctionInput); ok {
				return s.setupFunction(context, req)
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
