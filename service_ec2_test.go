package endly_test

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pkg/errors"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
)

func getInstanceStatus(awsCredential, instance string) (string, error) {
	manager := endly.NewManager()
	context := manager.NewContext(toolbox.NewContext())
	service, _ := context.Service(endly.Ec2ServiceID)
	serviceResponse := service.Run(context, &endly.EC2CallRequest{
		Credential: awsCredential,
		Method:     "DescribeInstances",
		Input: map[string]interface{}{
			"InstanceIds": []interface{}{
				instance,
			},
		},
	})
	if serviceResponse.Error != "" {
		return "", errors.New(serviceResponse.Error)
	}

	response, ok := serviceResponse.Response.(endly.EC2CallResponse)
	if !ok {
		return "", fmt.Errorf("expected endly.EC2CallResponse but had %T", serviceResponse.Response)
	}

	awsResponse, ok := response.(ec2.DescribeInstancesOutput)
	if !ok {
		return "", fmt.Errorf("expected %T but had %T", &ec2.DescribeInstancesOutput{}, response)
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
	manager := endly.NewManager()
	context := manager.NewContext(toolbox.NewContext())
	service, _ := context.Service(endly.Ec2ServiceID)
	serviceResponse := service.Run(context, &endly.EC2CallRequest{
		Credential: awsCredential,
		Method:     "StartInstances",
		Input: map[string]interface{}{
			"InstanceIds": []interface{}{
				instance,
			},
		},
	})
	if serviceResponse.Error != "" {
		return "", errors.New(serviceResponse.Error)
	}

	response, ok := serviceResponse.Response.(endly.EC2CallResponse)
	if !ok {
		return "", fmt.Errorf("expected endly.EC2CallResponse but had %T", serviceResponse.Response)
	}

	_, ok = response.(*ec2.StartInstancesOutput)
	if !ok {
		return "", fmt.Errorf("expected %T but had %T", &ec2.StartInstancesOutput{}, response)
	}
	return "", nil
}

func stopInstance(awsCredential, instance string) (string, error) {
	manager := endly.NewManager()
	context := manager.NewContext(toolbox.NewContext())
	service, _ := context.Service(endly.Ec2ServiceID)
	serviceResponse := service.Run(context, &endly.EC2CallRequest{
		Credential: awsCredential,
		Method:     "StopInstances",
		Input: map[string]interface{}{
			"InstanceIds": []interface{}{
				instance,
			},
		},
	})
	if serviceResponse.Error != "" {
		return "", errors.New(serviceResponse.Error)
	}

	response, ok := serviceResponse.Response.(endly.EC2CallResponse)
	if !ok {
		return "", fmt.Errorf("expected endly.EC2CallResponse,  but had %T", serviceResponse.Response)
	}

	_, ok = response.(*ec2.StopInstancesOutput)
	if !ok {
		return "", fmt.Errorf("expected %T but had %T", &ec2.StopInstancesOutput{}, response)
	}
	return "", nil
}
