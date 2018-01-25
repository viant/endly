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
	Method     string
	Parameters []interface{}
}

const (
	//Ec2ServiceID represents nop service id.
	Ec2ServiceID = "aws/ec2"

	//AwsEc2 represent asw ec2
	AwsEc2 = "AwsEc2"

	//Ec2ServiceCallAction represents run action
	Ec2ServiceCallAction = "call"
)

//Ec2 represent no operation
type Ec2 struct{}

//no operation service
type ec2Service struct {
	*AbstractService
}

func (s *ec2Service) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok", Response: request}
	var err error
	switch actualRequest := request.(type) {
	case *EC2CallRequest:
		response.Response, err = s.run(context, actualRequest)
	default:
		err = fmt.Errorf("unsupported request type: %T", request)
	}

	if err != nil {
		response.Status = "error"
		response.Error = fmt.Sprintf("%v", err)
	}

	defer s.End(context)(startEvent, Pairs("response", response))
	return response
}

func (s *ec2Service) NewRequest(action string) (interface{}, error) {
	if action == Ec2ServiceCallAction {
		return &EC2CallRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

func (s *ec2Service) NewResponse(action string) (interface{}, error) {
	if action == Ec2ServiceCallAction {
		return struct{}{}, nil
	}
	return s.AbstractService.NewResponse(action)
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

//NewEc2Service creates a new AWS Ec2 service.
func NewEc2Service() Service {
	var result = &ec2Service{
		AbstractService: NewAbstractService(Ec2ServiceID,
			Ec2ServiceCallAction),
	}
	result.AbstractService.Service = result
	return result
}
