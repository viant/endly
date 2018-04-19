package ec2

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/viant/toolbox/cred"
)

//GetAWSCredentialConfig returns *aws.Config for provided credential
func GetAWSCredentialConfig(config *cred.Config) (*aws.Config, error) {
	awsCredentials := credentials.NewStaticCredentials(config.Key, config.Secret, "")
	_, err := awsCredentials.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get aws credential: %v, %v", config.Key, err)
	}
	awsConfig := aws.NewConfig().WithRegion(config.Region).WithCredentials(awsCredentials)
	return awsConfig, nil
}

//GetEc2Client creates ec2 client for passed in credential file
func GetEc2Client(credConfig *cred.Config) (*ec2.EC2, error) {
	config, err := GetAWSCredentialConfig(credConfig)
	if err != nil {
		return nil, err
	}
	ec2Session := session.Must(session.NewSession())
	return ec2.New(ec2Session, config), nil
}
