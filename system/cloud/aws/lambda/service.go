package lambda

import (
	"encoding/json"
	"fmt"
	aaws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/pkg/errors"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/aws"
	"github.com/viant/endly/system/cloud/aws/cloudwatchevents"
	"github.com/viant/endly/system/cloud/aws/ec2"
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
		for {
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

func (s *service) call(context *endly.Context, request *CallInput) (*CallOutput, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	input := lambda.InvokeInput(*request)
	output, err := client.Invoke(&input)
	if err != nil {
		return nil, err
	}
	response := &CallOutput{
		InvokeOutput: output,
	}
	if len(output.Payload) > 0 {
		payloadText := toolbox.AsString(output.Payload)
		if toolbox.IsStructuredJSON(payloadText) {
			if err = json.Unmarshal(output.Payload, &response.Response); err == nil {
				output.Payload = nil
			}
		} else {
			response.Response = payloadText
			output.Payload = nil
		}
	}
	return response, err
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
		policy := &iam.PolicyDocument{}
		if err = json.Unmarshal([]byte(*getResponse.Policy), &policy); err != nil {
			return nil, err
		}
		if len(policy.Statement) > 0 {
			for _, statement := range policy.Statement {
				action, _ := statement.Action.Value()
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



func (s *service) deploy(context *endly.Context, request *DeployInput) (output *DeployOutput, err error) {
	output = &DeployOutput{}
	err = s.AbstractService.RunInBackground(context, func() error {
		output, err = s.deployFunctionInBackground(context, request)
		return err
	})
	return output, err
}


func (s *service) expand(context *endly.Context, values ...*string) {
	state := context.State()
	for i := range values {
		if values[i] == nil {
			continue
		}
		*values[i] = state.ExpandAsText(*values[i])
	}
}

func (s *service) deployFunctionInBackground(context *endly.Context, request *DeployInput) (*DeployOutput, error) {

	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	if request.VpcMatcher != nil {
		vpcOutput := &ec2.GetVpcConfigOutput{}
		if err = endly.Run(context, request.VpcMatcher, vpcOutput); err != nil {
			return nil, errors.Wrap(err, "failed to match vpc")
		}
		request.VpcConfig = &lambda.VpcConfig{
			SecurityGroupIds: vpcOutput.SecurityGroupIds,
			SubnetIds:        vpcOutput.SubnetIds,
		}
	}
	s.expand(context, request.FunctionName, request.RoleName, request.AssumeRolePolicyDocument)
	output := &DeployOutput{}
	output.RoleInfo = &iam.GetRoleInfoOutput{}
	if err = endly.Run(context, &request.SetupRolePolicyInput, &output.RoleInfo); err != nil {
		return nil, errors.Wrap(err, "failed to setup policy")
	}
	request.Role = output.RoleInfo.Role.Arn
	var functionConfig *lambda.FunctionConfiguration
	functionOutput, foundErr := client.GetFunction(&lambda.GetFunctionInput{
		FunctionName: request.FunctionName,
	})

	if foundErr == nil {
		functionConfig = functionOutput.Configuration


		if request.Schedule == nil && functionConfig != nil {
			if err = endly.Run(context, &cloudwatchevents.DeleteRuleInput{
				TargetArn: functionConfig.FunctionArn,
			}, nil); err != nil {
				return nil, errors.Wrap(err, "failed to delete schedule rule")
			}
		}
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
			return nil, errors.Wrap(err, "failed to update function")
		}

		if createFunction.Code.ZipFile != nil && !hasDataChanged(createFunction.Code.ZipFile, *functionConfig.CodeSha256) {
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
			return nil, errors.Wrap(err, "failed to update function code")
		}

	} else {
		if functionConfig, err = client.CreateFunction(&request.CreateFunctionInput); err != nil {
			return nil, errors.Wrap(err, "failed to create function")
		}
	}
	output.FunctionConfiguration = functionConfig
	if len(request.Triggers) > 0 {
		triggersOutput, err := s.setupTriggerSource(context, &SetupTriggerSourceInput{FunctionName: request.FunctionName, Triggers: request.Triggers})
		if err != nil {
			return nil, errors.Wrap(err, "failed to setup trigger")
		}
		output.EventMappings = triggersOutput.EventMappings
	} else {
		output.EventMappings, err = s.getEventSourceMappings(context, functionConfig.FunctionName)
		if err != nil {
			return nil, err
		}
	}

	if request.Schedule != nil {
		scheduleOutput := &cloudwatchevents.DeployRuleOutput{}
		scheduleRule :=  request.ScheduleDeployRule()
		if err = endly.Run(context, request.ScheduleDeployRule(), scheduleOutput); err != nil {
			return nil, errors.Wrapf(err, "failed to put schedule rule: %s", scheduleRule)
		}

		id, err := aws.NextID()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate id")
		}
		if _, err = s.setupPermission(context, &SetupPermissionInput{
			StatementId:&id,
			SourceArn:scheduleOutput.Rule.Arn,
			FunctionName: request.FunctionName,
			Action:aaws.String("lambda:InvokeFunction"),
			Principal:aaws.String("events.amazonaws.com"),
		});  err != nil {
			return nil, errors.Wrapf(err, "failed to add permission to %v", scheduleOutput.Rule.Arn)
		}



		scheduleEventInput, err := request.ScheduleEventsInput(scheduleOutput.Rule.Arn)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create put schedule event input %s", scheduleEventInput)
		}
		if err = endly.Run(context, scheduleEventInput, nil); err != nil {
			return nil, errors.Wrapf(err, "failed to put schedule event %s", scheduleEventInput)
		}

	}
	return output, err
}

func (s *service) getEventSourceMappings(context *endly.Context, functionName *string) ([]*lambda.EventSourceMappingConfiguration, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	var result = make([]*lambda.EventSourceMappingConfiguration, 0)
	var nextMarker *string
	for ;; {
		listOutput, err := client.ListEventSourceMappings(&lambda.ListEventSourceMappingsInput{
			FunctionName: functionName,
			Marker:nextMarker,
		})
		if err != nil {
			return nil, err
		}
		if listOutput != nil && len(listOutput.EventSourceMappings) > 0 {
			result = append(result, listOutput.EventSourceMappings...)
		}
		nextMarker = listOutput.NextMarker
		if nextMarker == nil {
			break
		}
	}
	return result, nil
}


/*
This method uses EventSourceMappingsInput, so only the following source are supported at the moment,

	//    * Amazon Kinesis - The ARN of the data stream or a stream consumer.
	//    * Amazon DynamoDB Streams - The ARN of the stream.
	//    * Amazon Simple Queue Service - The ARN of the queue.

*/
func (s *service) setupTriggerSource(context *endly.Context, request *SetupTriggerSourceInput) (*SetupTriggerSourceOutput, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	response := &SetupTriggerSourceOutput{
		EventMappings: make([]*lambda.EventSourceMappingConfiguration, 0),
	}

	for _, trigger := range request.Triggers {
		if trigger.SourceARN == nil {
			switch trigger.Type {
			case "sqs":
				trigger.SourceARN, err = aws.GetSqsURL(context, trigger.Source)
			case "kinesisStream":
				trigger.SourceARN, err = aws.GetKinesisStreamARN(context, trigger.Source)
			case "kinesisConsumer":
				trigger.SourceARN, err = aws.GetKinesisConsumerARN(context, trigger.Source)
			case "dynamodb":
				trigger.SourceARN, err = aws.GetDynamoDBTableARN(context, trigger.Source)
			}
			if err != nil {
				return nil, err
			}
		}
		listOutput, _ := client.ListEventSourceMappings(&lambda.ListEventSourceMappingsInput{
			FunctionName:   request.FunctionName,
			EventSourceArn: trigger.SourceARN,
		})
		if listOutput == nil || len(listOutput.EventSourceMappings) == 0 {
			mappingConfig, err := client.CreateEventSourceMapping(&lambda.CreateEventSourceMappingInput{
				Enabled:                   trigger.Enabled,
				BatchSize:                 trigger.BatchSize,
				EventSourceArn:            trigger.SourceARN,
				FunctionName:              request.FunctionName,
				StartingPosition:          trigger.StartingPosition,
				StartingPositionTimestamp: trigger.StartingPositionTimestamp,
			})
			if err != nil {
				return nil, err
			}
			response.EventMappings = append(response.EventMappings, mappingConfig)
			continue
		}
		for _, eventMapping := range listOutput.EventSourceMappings {
			mappingConfig, err := client.UpdateEventSourceMapping(&lambda.UpdateEventSourceMappingInput{
				UUID:         eventMapping.UUID,
				Enabled:      trigger.Enabled,
				BatchSize:    trigger.BatchSize,
				FunctionName: request.FunctionName,
			})
			if err != nil {
				return nil, err
			}
			response.EventMappings = append(response.EventMappings, mappingConfig)
		}
	}
	return response, nil
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
		Action: "deploy",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "deploy", &DeployInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &DeployOutput{}),
		},
		RequestProvider: func() interface{} {
			return &DeployInput{}
		},
		ResponseProvider: func() interface{} {
			return &DeployOutput{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*DeployInput); ok {
				return s.deploy(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "setupTriggerSource",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "setupTriggerSource", &SetupTriggerSourceInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &SetupTriggerSourceOutput{}),
		},
		RequestProvider: func() interface{} {
			return &SetupTriggerSourceInput{}
		},
		ResponseProvider: func() interface{} {
			return &SetupTriggerSourceOutput{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*SetupTriggerSourceInput); ok {
				return s.setupTriggerSource(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "call",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "call", &CallInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &CallOutput{}),
		},
		RequestProvider: func() interface{} {
			return &CallInput{}
		},
		ResponseProvider: func() interface{} {
			return &CallOutput{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*CallInput); ok {
				response, err := s.call(context, req)
				if err == nil {
					context.Publish(aws.NewOutputEvent("call", "lambda", response))
				}
				return response, err
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
