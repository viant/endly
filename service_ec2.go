package endly

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/url"
)

//EC2Call represents ec2 call.
type EC2Call struct {
	Method     string        `required:"true" description:"ec2 client method name"`
	Parameters []interface{} `required:"true" description:"ec2 client method paramters"`
}

const (
	//Ec2ServiceID represents nop service id.
	Ec2ServiceID = "aws/ec2"
)

//Ec2 represent no operation
type Ec2 struct{}

//no operation service
type ec2Service struct {
	*AbstractService
}

//GetAWSCredentialConfig returns *aws.Config for provided credential
func GetAWSCredentialConfig(credential string) (*aws.Config, error) {
	config := &cred.Config{}
	resource := url.NewResource(credential)
	err := resource.JSONDecode(config)
	if err != nil {
		return nil, fmt.Errorf("failed to load aws credential: %v", credential)
	}
	awsCredentials := credentials.NewStaticCredentials(config.Key, config.Secret, "")
	_, err = awsCredentials.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get aws credential: %v, %v", credential, err)
	}
	awsConfig := aws.NewConfig().WithRegion(config.Region).WithCredentials(awsCredentials)
	return awsConfig, nil
}

//GetEc2Client creates ec2 client for passed in credential file
func GetEc2Client(credential string) (*ec2.EC2, error) {
	config, err := GetAWSCredentialConfig(credential)
	if err != nil {
		return nil, err
	}
	ec2Session := session.Must(session.NewSession())
	return ec2.New(ec2Session, config), nil
}

func (s *ec2Service) run(context *Context, request *EC2CallRequest) (EC2CallResponse, error) {
	client, err := GetEc2Client(request.Credential)
	if err != nil {
		return nil, err
	}

	return s.call(context, client, request)
}

func (s *ec2Service) call(context *Context, caller interface{}, request *EC2CallRequest) (callResponse EC2CallResponse, err error) {
	call := request.AsEc2Call()
	callResponse, err = s.callMethod(caller, call.Method, call.Parameters)
	if err != nil {
		return nil, err
	}
	return callResponse, err
}

func (s *ec2Service) callMethod(owner interface{}, methodName string, parameters []interface{}) (EC2CallResponse, error) {
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
  "Credential": "${env.HOME}/.secret/aws-west.json",
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
        "Secret":"SSSSSSSSS"
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
		"Credential": "${env.HOME}/.secret/aws-west.json",
		"Method": "StartInstances",
		"Input": {
			"InstanceIds": [
				"i-*********"
			]
		}
	}`
)

func (s *ec2Service) registerRoutes() {
	s.Register(&ServiceActionRoute{
		Action: "call",
		RequestInfo: &ActionInfo{
			Description: "call proxies request into github.com/aws/aws-sdk-go/service/ec2.EC2 client",
			Examples: []*ExampleUseCase{
				{
					UseCase: "credential file",
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
		ResponseInfo: &ActionInfo{
			Description: "response from github.com/aws/aws-sdk-go/service/ec2.EC2 client",
			Examples: []*ExampleUseCase{
				{
					UseCase: "get instance status",
					Data:    ex2GetInstanceResponseExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &EC2CallRequest{}
		},
		ResponseProvider: func() interface{} {
			return struct{}{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*EC2CallRequest); ok {
				return s.run(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//NewEc2Service creates a new AWS Ec2 service.
func NewEc2Service() Service {
	var result = &ec2Service{
		AbstractService: NewAbstractService(Ec2ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
