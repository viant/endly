package endly_test

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"os"
	"path"
	"testing"
)

func getInstanceStatus(awsCredential, instance string) (string, error) {
	manager := endly.NewManager()
	context := manager.NewContext(toolbox.NewContext())
	service, _ := context.Service(endly.Ec2ServiceID)
	serviceResponse := service.Run(context, &endly.Ec2CallRequest{
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

	response, ok := serviceResponse.Response.(*endly.Ec2CallResponse)
	if !ok {
		return "", fmt.Errorf("expected %T but had %T", &endly.Ec2CallResponse{}, serviceResponse.Response)
	}

	awsResponse, ok := response.Response.(*ec2.DescribeInstancesOutput)
	if !ok {
		return "", fmt.Errorf("expected %T but had %T", &ec2.DescribeInstancesOutput{}, response.Response)
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
	serviceResponse := service.Run(context, &endly.Ec2CallRequest{
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

	response, ok := serviceResponse.Response.(*endly.Ec2CallResponse)
	if !ok {
		return "", fmt.Errorf("expected %T but had %T", &endly.Ec2CallResponse{}, serviceResponse.Response)
	}

	_, ok = response.Response.(*ec2.StartInstancesOutput)
	if !ok {
		return "", fmt.Errorf("expected %T but had %T", &ec2.StartInstancesOutput{}, response.Response)
	}
	return "", nil
}


func stopInstance(awsCredential, instance string) (string, error) {
	manager := endly.NewManager()
	context := manager.NewContext(toolbox.NewContext())
	service, _ := context.Service(endly.Ec2ServiceID)
	serviceResponse := service.Run(context, &endly.Ec2CallRequest{
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

	response, ok := serviceResponse.Response.(*endly.Ec2CallResponse)
	if !ok {
		return "", fmt.Errorf("expected %T but had %T", &endly.Ec2CallResponse{}, serviceResponse.Response)
	}

	_, ok = response.Response.(*ec2.StopInstancesOutput)
	if !ok {
		return "", fmt.Errorf("expected %T but had %T", &ec2.StopInstancesOutput{}, response.Response)
	}
	return "", nil
}

func TestEc2Service_Run(t *testing.T) {

	os.Setenv("awsTestInstanceId", "i-0ef8d9260eaf47fdf")
	awsCredential := path.Join(os.Getenv("HOME"), ".secret/aws.json")
	if toolbox.FileExists(awsCredential) {
		var testInstanceId = os.Getenv("awsTestInstanceId")
		status, err := getInstanceStatus(awsCredential, testInstanceId)
		if assert.Nil(t, err) {
			if status == "stopped" {
				startInstance(awsCredential, testInstanceId)
			} else {
				stopInstance(awsCredential, testInstanceId)
			}

		}//use WorkflowRepeatAction Request
		stopInstance(awsCredential, testInstanceId)
	}

}
