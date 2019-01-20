package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/go-errors/errors"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/cred"
	"reflect"
	"strings"
)

var configKey = (*aws.Config)(nil)

//GetAWSCredentialConfig returns *aws.Config for provided credential
func GetAWSCredentialConfig(config *cred.Config) (*aws.Config, error) {
	awsCredentials := credentials.NewStaticCredentials(config.Key, config.Secret, "")
	_, err := awsCredentials.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get aws credential: %v, %v", config.Key, err)
	}

	awsConfig := aws.NewConfig().WithRegion(config.Region).WithCredentials(awsCredentials)
	if config.AccountID == "" {
		iamSession := session.Must(session.NewSession())
		iamClient := iam.New(iamSession, awsConfig)
		output, err := iamClient.GetUser(&iam.GetUserInput{})
		if err != nil {
			return nil, err
		}
		if output.User.Arn != nil {
			config.AccountID = strings.Split(*output.User.Arn, ":")[4]
		}
	}
	return awsConfig, nil
}


//InitCredentials get or creates aws credential config
func InitCredentials(context *endly.Context, rawRequest map[string]interface{}, key interface{}) (*aws.Config, error) {
	if len(rawRequest) == 0 {
		return nil, fmt.Errorf("request was empty")
	}
	secrets := &struct {
		Credentials string
	}{}
	if err := toolbox.DefaultConverter.AssignConverted(secrets, rawRequest); err != nil {
		return nil, err
	}
	if secrets.Credentials == "" {
		if context.Contains(key) {
			return nil, nil
		}
		if context.Contains(configKey) {
			awsConfig := &aws.Config{}
			if context.GetInto(configKey, &awsConfig) {
				return awsConfig, nil
			}
		}
		return nil, fmt.Errorf("unable to create clinet %T, credentials attribute was empty", key)
	}
	config, err := context.Secrets.GetCredentials(secrets.Credentials)
	if err != nil {
		return nil, err
	}
	if context.Contains(key) {
		context.Remove(key)
	}
	if context.Contains(configKey) {
		context.Remove(configKey)
	}

	awsConfig, err := GetAWSCredentialConfig(config)
	if err != nil {
		return nil, err
	}
	_ = context.Put(configKey, awsConfig)
	return awsConfig, err
}

//GetClient get or creates aws client
func GetClient(context *endly.Context, provider interface{}, client interface{}) error {
	if !context.Contains(configKey) {
		return errors.New("unable to lookup aws.Config")
	}
	awsConfig := &aws.Config{}
	if !context.GetInto(configKey, &awsConfig) {
		return errors.New("unable to fetch aws.Config")
	}
	sess := session.Must(session.NewSession())
	output := toolbox.CallFunction(provider, sess, awsConfig)
	//TODO safety check
	reflect.ValueOf(client).Elem().Set(reflect.ValueOf(output[0]))
	return nil
}
