package ec2

import "github.com/aws/aws-sdk-go/service/ec2"

//Filter represents a filter
type Filter struct {
	ID    string `description:"if specified ID match"`
	VpcID string
	Name  string            `description:"name is a tags.Name helper"`
	Tags  map[string]string `description:"matching tags"`
}

//GetVpcInput represents vpc request
type GetVpcInput struct {
	Filter
}

//GetVpcInput represents vpc response
type GetVpcOutput struct {
	*ec2.Vpc
}

//GetInstanceInput represents get instance request
type GetInstanceInput struct {
	Filter
}

//GetInstanceInput represents get instance response
type GetInstanceOutput struct {
	*ec2.Instance
}

//GetVpcConfigInput represents get vpc config request for iter vpc or instance name
type GetVpcConfigInput struct {
	Vpc      *Filter
	Instance *Filter
}


//GetSecurityGroupInput represents request
type GetSecurityGroupInput struct {
	Filter
}

//GetSecurityGroupsOutput represents response
type GetSecurityGroupsOutput struct {
	Groups []*ec2.SecurityGroup
}

//GetSubnetsInput represents request
type GetSubnetsInput struct {
	Filter
}

//GetSubnetsOutput represents response
type GetSubnetsOutput struct {
	Subnets []*ec2.Subnet
}


//GetVpcConfigInput represents get vpc config response
type GetVpcConfigOutput struct {
	VpcID *string
	// A list of VPC security groups IDs.
	SecurityGroupIds []*string `type:"list"`

	// A list of VPC subnet IDs.
	SubnetIds []*string `type:"list"`
}

//Init initializes request
func (i *GetVpcConfigInput) Init() error {
	if i.Instance != nil {
		if err := i.Instance.Init(); err != nil {
			return err
		}
	}
	if i.Vpc != nil {
		if err := i.Vpc.Init(); err != nil {
			return err
		}
	}
	return nil
}

//Init initialises filter tags
func (f *Filter) Init() error {
	if len(f.Tags) == 0 {
		f.Tags = make(map[string]string)
	}
	if f.Name != "" {
		f.Tags["Name"] = f.Name
	}
	return nil
}
