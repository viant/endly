package lambda

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/go-errors/errors"
	ciam "github.com/viant/endly/system/cloud/aws/iam"
	"time"
)

//RecreateFunctionInput drops function if exist to create a new one
type RecreateFunctionInput lambda.CreateFunctionInput

//DropFunctionInput remove a function with all dependencies
type DropFunctionInput lambda.DeleteFunctionInput

type EventSourceMapping struct {
	Source                    string
	Type                      string
	SourceARN                 *string
	Enabled                   *bool
	BatchSize                 *int64     `min:"1" type:"integer"`
	StartingPosition          *string    `type:"string" enum:"EventSourcePosition"`
	StartingPositionTimestamp *time.Time `type:"timestamp"`
}

//DeployInput setup function, creates or updates existing one
type DeployInput struct {
	lambda.CreateFunctionInput `yaml:",inline" json:",inline"`
	ciam.SetupRolePolicyInput `yaml:",inline" json:",inline"`
	Triggers []*EventSourceMapping
}

type DeployOutput struct {
	*lambda.FunctionConfiguration `json:",inline"`
	RoleInfo      *ciam.GetRoleInfoOutput
	EventMappings []*lambda.EventSourceMappingConfiguration
}

//SetupPermissionInput creates a permission if it does not exists
type SetupPermissionInput lambda.AddPermissionInput

//SetupTriggerSourceInput represents setup triggers input
type SetupTriggerSourceInput struct {
	FunctionName *string
	Triggers     []*EventSourceMapping
}

//SetupTriggerSourceOutput represents  setup triggers output
type SetupTriggerSourceOutput struct {
	EventMappings []*lambda.EventSourceMappingConfiguration
}

//CallInput represents a call request
type CallInput lambda.InvokeInput

//CallOutput represents a call response
type CallOutput struct {
	*lambda.InvokeOutput
	Response interface{}
}

func (i *DeployInput) Init() error {
	if i.DefaultPolicyDocument == nil {
		policyDocument := string(DefaultTrustPolicy)
		i.DefaultPolicyDocument = &policyDocument
	}
	return nil
}

func (i *DeployInput) Validate() error {
	if i.CreateFunctionInput.FunctionName == nil {
		return fmt.Errorf("functionName was empty")
	}
	if i.CreateFunctionInput.Code == nil {
		return fmt.Errorf("code was empty")
	}
	if i.SetupRolePolicyInput.RoleName == nil {
		return fmt.Errorf("roleName was empty")
	}

	return nil
}

func (i *SetupTriggerSourceInput) Validate() error {
	if i.FunctionName == nil {
		return errors.New("functionName was empty")
	}
	if len(i.Triggers) == 0 {
		return errors.New("triggers were empty")
	}
	for _, trigger := range i.Triggers {
		switch trigger.Type {
		case "sqs", "kinesisStream", "kinesisConsumer", "dynamodb":
		default:
			return fmt.Errorf("unsupported source type: %v, supported: sqs,kinesisStream,kinesisConsumer,dynamodb", trigger.Type)
		}
		if trigger.Source == "" {
			return errors.New("source was empty")
		}
	}
	return nil
}
