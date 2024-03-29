package ec2

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/viant/endly"
	"github.com/viant/endly/service/system/cloud/aws"
	"log"
)

const (
	//ServiceID aws iam service id.
	ServiceID = "aws/ec2"
)

// no operation service
type service struct {
	*endly.AbstractService
}

func (s *service) registerRoutes() {
	client := &ec2.EC2{}
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
		Action: "getVpcConfig",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "getVpcConfig", &GetVpcConfigInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &GetVpcConfigOutput{}),
		},
		RequestProvider: func() interface{} {
			return &GetVpcConfigInput{}
		},
		ResponseProvider: func() interface{} {
			return &GetVpcConfigOutput{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*GetVpcConfigInput); ok {
				return s.getVpcConfig(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "getVpc",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "getVpc", &GetVpcInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &GetVpcOutput{}),
		},
		RequestProvider: func() interface{} {
			return &GetVpcInput{}
		},
		ResponseProvider: func() interface{} {
			return &GetVpcOutput{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*GetVpcInput); ok {
				return s.getVpc(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
	s.Register(&endly.Route{
		Action: "getInstance",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "getInstance", &GetInstanceInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &GetInstanceOutput{}),
		},
		RequestProvider: func() interface{} {
			return &GetInstanceInput{}
		},
		ResponseProvider: func() interface{} {
			return &GetInstanceOutput{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*GetInstanceInput); ok {
				return s.getInstance(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "getSecurityGroups",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "getSecurityGroups", &GetSecurityGroupInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &GetSecurityGroupsOutput{}),
		},
		RequestProvider: func() interface{} {
			return &GetSecurityGroupInput{}
		},
		ResponseProvider: func() interface{} {
			return &GetSecurityGroupsOutput{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*GetSecurityGroupInput); ok {
				return s.getSecurityGroups(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "getSubnets",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "getSubnets", &GetSubnetsInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &GetSubnetsOutput{}),
		},
		RequestProvider: func() interface{} {
			return &GetSubnetsInput{}
		},
		ResponseProvider: func() interface{} {
			return &GetSubnetsOutput{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*GetSubnetsInput); ok {
				return s.getSubnets(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "getVpcConfig",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "getVpcConfig", &GetVpcConfigInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &GetVpcConfigOutput{}),
		},
		RequestProvider: func() interface{} {
			return &GetVpcConfigInput{}
		},
		ResponseProvider: func() interface{} {
			return &GetVpcConfigOutput{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*GetVpcConfigInput); ok {
				return s.getVpcConfig(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

func (s *service) getVpcConfig(context *endly.Context, input *GetVpcConfigInput) (*GetVpcConfigOutput, error) {
	output := &GetVpcConfigOutput{
		SecurityGroupIds: make([]*string, 0),
		SubnetIds:        make([]*string, 0),
	}

	if input.Vpc != nil {
		getVpcOutput, err := s.getVpc(context, &GetVpcInput{*input.Vpc})
		if err != nil {
			return nil, err
		}
		if getVpcOutput.Vpc == nil || getVpcOutput.VpcId == nil {
			return nil, fmt.Errorf("failed to lookup vpc %v", input.Vpc)
		}
		output.VpcID = getVpcOutput.VpcId
	}

	if input.Instance != nil {
		getInstanceOutput, err := s.getInstance(context, &GetInstanceInput{*input.Instance})
		if err != nil {
			return nil, err
		}
		if getInstanceOutput.Instance == nil || getInstanceOutput.InstanceId == nil {
			return nil, fmt.Errorf("failed to lookup instance %v", input.Instance)
		}
		if len(getInstanceOutput.NetworkInterfaces) == 0 {
			return nil, fmt.Errorf("instance %v, does not have network instance", getInstanceOutput.InstanceId)
		}

		if getInstanceOutput.VpcId != nil {
			output.VpcID = getInstanceOutput.VpcId
		}
		if output.VpcID == nil {
			for _, network := range getInstanceOutput.NetworkInterfaces {
				if network.VpcId != nil {
					output.VpcID = network.VpcId
					break
				}
			}
		}
	}
	if output.VpcID == nil {
		return nil, fmt.Errorf("failed to find vpc, %v", input)
	}
	subnetInput := &GetSubnetsInput{
		Filter: Filter{VpcID: *output.VpcID},
	}

	if input.Instance != nil {
		subnetInput.SubnetExclusionTags = input.Instance.SubnetExclusionTags
	}
	if input.Vpc != nil {
		subnetInput.SubnetExclusionTags = input.Vpc.SubnetExclusionTags
	}

	subnetsOutput, err := s.getSubnets(context, subnetInput)
	if err != nil {
		return nil, err
	}
	if subnetsOutput == nil || len(subnetsOutput.Subnets) == 0 {
		return nil, fmt.Errorf("failed to locate subnet for vpc: %v", output.VpcID)
	}
	for _, subnet := range subnetsOutput.Subnets {
		output.SubnetIds = append(output.SubnetIds, subnet.SubnetId)
	}

	securityGroupsOuput, err := s.getSecurityGroups(context, &GetSecurityGroupInput{
		Filter: Filter{
			VpcID: *output.VpcID,
		},
	})
	if securityGroupsOuput == nil || len(securityGroupsOuput.Groups) == 0 {
		return nil, fmt.Errorf("failed to lookup securiry group for vpc: %v", output.VpcID)
	}
	for _, group := range securityGroupsOuput.Groups {
		output.SecurityGroupIds = append(output.SecurityGroupIds, group.GroupId)
	}
	return output, nil
}

func (s *service) matchVpc(output *ec2.DescribeVpcsOutput, filter *Filter) *ec2.Vpc {
	if filter.VpcID != "" {
		filter.ID = filter.VpcID
	}
	for _, candidate := range output.Vpcs {
		if filter.ID != "" && *candidate.VpcId == filter.ID {
			return candidate
		}
		if matchesByTags(filter.Tags, candidate.Tags) {
			return candidate
		}
	}
	return nil
}

func (s *service) matchInstance(output *ec2.DescribeInstancesOutput, filter *Filter) *ec2.Instance {
	for _, candidate := range output.Reservations {
		if len(candidate.Instances) == 0 {
			continue
		}
		for _, candidate := range candidate.Instances {
			if candidate.VpcId != nil && filter.VpcID != "" && *candidate.VpcId == filter.VpcID {
				return candidate
			}
			if filter.ID != "" && *candidate.InstanceId == filter.ID {
				return candidate
			}
			if matchesByTags(filter.Tags, candidate.Tags) {
				return candidate
			}
		}
	}
	return nil
}

func (s *service) matchSecurityGroups(output *ec2.DescribeSecurityGroupsOutput, filter *Filter, matched *[]*ec2.SecurityGroup) {
	for _, candidate := range output.SecurityGroups {
		if candidate.VpcId != nil && filter.VpcID != "" && *candidate.VpcId == filter.VpcID {
			*matched = append(*matched, candidate)
		}
		if filter.ID != "" && *candidate.GroupId == filter.ID {
			*matched = append(*matched, candidate)
		}
		if matchesByTags(filter.Tags, candidate.Tags) {
			*matched = append(*matched, candidate)
		}
	}
}

func (s *service) matchSubnets(output *ec2.DescribeSubnetsOutput, filter *Filter, matched *[]*ec2.Subnet) {
	for _, candidate := range output.Subnets {
		if len(filter.SubnetExclusionTags) > 0 {
			if matchesByTags(filter.SubnetExclusionTags, candidate.Tags) {
				continue
			}
		}
		if candidate.VpcId != nil && filter.VpcID != "" && *candidate.VpcId == filter.VpcID {
			*matched = append(*matched, candidate)
		}
		if filter.ID != "" && *candidate.SubnetId == filter.ID {
			*matched = append(*matched, candidate)
		}
		if matchesByTags(filter.Tags, candidate.Tags) {
			*matched = append(*matched, candidate)
		}
	}
}

func (s *service) getVpc(context *endly.Context, input *GetVpcInput) (*GetVpcOutput, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	output, err := client.DescribeVpcs(&ec2.DescribeVpcsInput{})
	if err != nil {
		return nil, err
	}
	vpc := s.matchVpc(output, &input.Filter)
	return &GetVpcOutput{
		Vpc: vpc,
	}, nil
}

func (s *service) getInstance(context *endly.Context, input *GetInstanceInput) (*GetInstanceOutput, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	output := &GetInstanceOutput{}
	var nextToken *string
	for {
		describeOutput, err := client.DescribeInstances(&ec2.DescribeInstancesInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}
		if output.Instance = s.matchInstance(describeOutput, &input.Filter); output.Instance != nil {
			break
		}

		if nextToken = describeOutput.NextToken; nextToken == nil {
			break
		}
	}
	return output, nil
}

func (s *service) getSecurityGroups(context *endly.Context, input *GetSecurityGroupInput) (*GetSecurityGroupsOutput, error) {
	output := &GetSecurityGroupsOutput{
		Groups: make([]*ec2.SecurityGroup, 0),
	}
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	var nextToken *string
	for {
		describeOutput, err := client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}
		s.matchSecurityGroups(describeOutput, &input.Filter, &output.Groups)
		if nextToken = describeOutput.NextToken; nextToken == nil {
			break
		}
	}
	return output, nil
}

func (s *service) getSubnets(context *endly.Context, input *GetSubnetsInput) (*GetSubnetsOutput, error) {
	output := &GetSubnetsOutput{
		Subnets: make([]*ec2.Subnet, 0),
	}
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	describeOutput, err := client.DescribeSubnets(&ec2.DescribeSubnetsInput{})
	if err != nil {
		return nil, err
	}
	s.matchSubnets(describeOutput, &input.Filter, &output.Subnets)

	return output, nil
}

// New creates a new EC2 service.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
