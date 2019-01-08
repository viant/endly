package ec2_test

import (
	"fmt"
	cec2 "github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/ec2"
	"github.com/viant/toolbox"
	"testing"
)

func getInstanceStatus(awsCredential, instance string) (string, error) {
	manager := endly.New()
	context := manager.NewContext(toolbox.NewContext())
	service, _ := context.Service(ec2.ServiceID)
	serviceResponse := service.Run(context, &ec2.CallRequest{
		Credentials: awsCredential,
		Method:      "DescribeInstances",
		Input: map[string]interface{}{
			"InstanceIds": []interface{}{
				instance,
			},
		},
	})
	if serviceResponse.Error != "" {
		return "", errors.New(serviceResponse.Error)
	}

	response, ok := serviceResponse.Response.(ec2.CallResponse)
	if !ok {
		return "", fmt.Errorf("expected endly.CallResponse but had %T", serviceResponse.Response)
	}

	awsResponse, ok := response.(cec2.DescribeInstancesOutput)
	if !ok {
		return "", fmt.Errorf("expected %T but had %T", &cec2.DescribeInstancesOutput{}, response)
	}

	if len(awsResponse.Reservations) > 0 {
		if len(awsResponse.Reservations[0].Instances) > 0 {
			var instance = awsResponse.Reservations[0].Instances[0]
			if instance.State != nil && instance.State.Name != nil {
				return *instance.State.Name, nil
			}

		}
	}
	return "", nil
}

func startInstance(awsCredential, instance string) (string, error) {
	manager := endly.New()
	context := manager.NewContext(toolbox.NewContext())
	service, _ := context.Service(ec2.ServiceID)
	serviceResponse := service.Run(context, &ec2.CallRequest{
		Credentials: awsCredential,
		Method:      "StartInstances",
		Input: map[string]interface{}{
			"InstanceIds": []interface{}{
				instance,
			},
		},
	})
	if serviceResponse.Error != "" {
		return "", errors.New(serviceResponse.Error)
	}

	response, ok := serviceResponse.Response.(ec2.CallResponse)
	if !ok {
		return "", fmt.Errorf("expected endly.CallResponse but had %T", serviceResponse.Response)
	}

	_, ok = response.(*cec2.StartInstancesOutput)
	if !ok {
		return "", fmt.Errorf("expected %T but had %T", &cec2.StartInstancesOutput{}, response)
	}
	return "", nil
}

func stopInstance(awsCredential, instance string) (string, error) {
	manager := endly.New()
	context := manager.NewContext(toolbox.NewContext())
	service, _ := context.Service(ec2.ServiceID)
	serviceResponse := service.Run(context, &ec2.CallRequest{
		Credentials: awsCredential,
		Method:      "StopInstances",
		Input: map[string]interface{}{
			"InstanceIds": []interface{}{
				instance,
			},
		},
	})
	if serviceResponse.Error != "" {
		return "", errors.New(serviceResponse.Error)
	}

	response, ok := serviceResponse.Response.(ec2.CallResponse)
	if !ok {
		return "", fmt.Errorf("expected endly.CallResponse,  but had %T", serviceResponse.Response)
	}

	_, ok = response.(*cec2.StopInstancesOutput)
	if !ok {
		return "", fmt.Errorf("expected %T but had %T", &cec2.StopInstancesOutput{}, response)
	}
	return "", nil
}

func Test_EC2CallRequest_AsEc2Call(t *testing.T) {

	{
		request := &ec2.CallRequest{
			Credentials: "abc",
			Method:      "Stop",
			Input: map[string]interface{}{
				"k1": 1,
			},
		}
		var ec2Call = request.AsCall()
		assert.NotNil(t, ec2Call)
		assert.NotNil(t, 1, len(ec2Call.Parameters))

	}
	{
		request := &ec2.CallRequest{
			Credentials: "abc",
			Method:      "Stop",
			Input:       nil,
		}
		var ec2Call = request.AsCall()
		assert.NotNil(t, ec2Call)
		assert.NotNil(t, 0, len(ec2Call.Parameters))

	}
}
