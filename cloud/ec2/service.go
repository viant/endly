package ec2

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
)

const (
	//ServiceID represents nop service id.
	ServiceID = "aws/ec2"
)

//no operation service
type service struct {
	*endly.AbstractService
}

func (s *service) run(context *endly.Context, request *CallRequest) (CallResponse, error) {
	client, err := GetEc2Client(request.Credentials)
	if err != nil {
		return nil, err
	}

	return s.call(context, client, request)
}

func (s *service) call(context *endly.Context, caller interface{}, request *CallRequest) (callResponse CallResponse, err error) {
	call := request.AsCall()
	callResponse, err = s.callMethod(caller, call.Method, call.Parameters)
	if err != nil {
		return nil, err
	}
	return callResponse, err
}

func (s *service) callMethod(owner interface{}, methodName string, parameters []interface{}) (CallResponse, error) {
	method, err := toolbox.GetFunction(owner, methodName)
	if err != nil {
		return nil, err
	}
	parameters, err = toolbox.AsCompatibleFunctionParameters(method, parameters)
	if err != nil {
		return nil, err
	}
	var response interface{}
	result := toolbox.CallFunction(method, parameters...)
	if len(result) == 2 {
		response = result[0]
		if result[1] != nil {
			if e, ok := result[1].(error); ok {
				return nil, e
			}
		}
	}
	return response, nil
}

const (
	ec2GetInstanceStatusExample = `{
  "Credentials": "${env.HOME}/.secret/aws-west.json",
  "Method": "DescribeInstances",
  "Input": {
    "InstanceIds": [
      "i-0139209d53****"
    ]
  }
}`

	ec2CredentialExample = `{
        "Region":"us-west-2",
        "Key":"KKKKKKKKK",
        "Secrets":"SSSSSSSSS"
}`

	ex2GetInstanceResponseExample = `{
  "Reservations": [
    {
      "Groups": null,
      "Instances": [
        {
          "AmiLaunchIndex": 0,
          "Architecture": "x86_64",
          "BlockDeviceMappings": [
            {
              "DeviceName": "/dev/sda1",
              "Ebs": {
                "DeleteOnTermination": true,
                "Status": "attached"
              }
            }
          ],
          "ClientToken": "",
          "EbsOptimized": false,
          "ElasticGpuAssociations": null,
          "EnaSupport": false,
          "Hypervisor": "xen",
          "IamInstanceProfile": null,
          "InstanceLifecycle": null,
          "InstanceType": "c4.2xlarge",
          "KernelId": null,
          "KeyName": "production",
          "Monitoring": {
            "State": "disabled"
          },
          "NetworkInterfaces": [
            {
              "Association": null,
              "Attachment": {
                "DeleteOnTermination": true,
                "DeviceIndex": 0,
                "Status": "attached"
              },
              "Description": "",
              "SourceDestCheck": true,
              "Status": "in-use",
            }
          ],
          "Placement": {
            "Affinity": null,
            "AvailabilityZone": "us-west-2b",
            "GroupName": "",
            "HostId": null,
            "SpreadDomain": null,
            "Tenancy": "default"
          },
          "Platform": null,
          "PublicDnsName": "",
          "PublicIpAddress": null,
          "RamdiskId": null,
          "RootDeviceName": "/dev/sda1",
          "RootDeviceType": "ebs",
          "SourceDestCheck": true,
          "SpotInstanceRequestId": null,
          "SriovNetSupport": null,
          "State": {
            "Code": 80,
            "Name": "stopped"
          }
        }
      ]
    }
  ]
}`

	ec2StartInstanceExample = `{
		"Credentials": "${env.HOME}/.secret/aws-west.json",
		"Method": "StartInstances",
		"Input": {
			"InstanceIds": [
				"i-*********"
			]
		}
	}`
)

func (s *service) registerRoutes() {
	s.Register(&endly.ServiceActionRoute{
		Action: "call",
		RequestInfo: &endly.ActionInfo{
			Description: "call proxies request into github.com/aws/aws-sdk-go/service/ec2.EC2 client",
			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "credentials file",
					Data:    ec2CredentialExample,
				},
				{
					UseCase: "get instance status",
					Data:    ec2GetInstanceStatusExample,
				},
				{
					UseCase: "start instance",
					Data:    ec2StartInstanceExample,
				},
			},
		},
		ResponseInfo: &endly.ActionInfo{
			Description: "response from github.com/aws/aws-sdk-go/service/ec2.EC2 client",
			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "get instance status",
					Data:    ex2GetInstanceResponseExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &CallRequest{}
		},
		ResponseProvider: func() interface{} {
			return struct{}{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*CallRequest); ok {
				return s.run(context, req)
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
