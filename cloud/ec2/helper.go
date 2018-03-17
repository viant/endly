package ec2

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/url"
)

//GetAWSCredentialConfig returns *aws.Config for provided credential
func GetAWSCredentialConfig(credential string) (*aws.Config, error) {
	config := &cred.Config{}
	resource := url.NewResource(credential)
	err := resource.Decode(config)
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
