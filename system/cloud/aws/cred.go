package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/go-errors/errors"
	"github.com/viant/endly"
	"github.com/viant/scy/cred"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"os"
	"reflect"
	"strings"
)

var configKey = (*aws.Config)(nil)

// GetAWSCredentialConfig returns *aws.Config for provided credential
func GetAWSCredentialConfig(config *cred.Generic) (*aws.Config, error) {
	awsCredentials := credentials.NewStaticCredentials(config.Key, config.Secret, "")
	_, err := awsCredentials.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get aws credential: %v, %v", config.Key, err)
	}

	if err != nil {
		return nil, err
	}

	awsConfig := aws.NewConfig().WithRegion(config.Region).WithCredentials(awsCredentials)
	if config.Id == "" {
		iamSession := session.Must(session.NewSession())
		iamClient := iam.New(iamSession, awsConfig)
		output, err := iamClient.GetUser(&iam.GetUserInput{})
		if err != nil {
			return nil, err
		}
		if output.User.Arn != nil {
			config.Id = strings.Split(*output.User.Arn, ":")[4]
		}
	}
	return awsConfig, nil
}

// InitCredentials get or creates aws credential config
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
	generic, err := context.Secrets.GetCredentials(context.Background(), secrets.Credentials)
	if err != nil {
		return nil, err
	}
	if context.Contains(key) {
		context.Remove(key)
	}
	if context.Contains(configKey) {
		context.Remove(configKey)
	}

	awsCred, err := GetAWSCredentialConfig(generic)
	if err != nil {
		return nil, err
	}
	credSession := generic.Session
	if credSession != nil && credSession.RoleArn != "" {
		region := generic.Region
		if region == "" {
			region = os.Getenv("AWS_REGION")
		}
		sess, err := session.NewSession(&aws.Config{
			Region:      &region,
			Credentials: credentials.NewStaticCredentials(generic.Key, generic.Secret, ""),
		})
		if err != nil {
			return nil, err
		}
		svc := sts.New(sess)
		result, err := svc.AssumeRole(&sts.AssumeRoleInput{
			RoleArn:         &credSession.RoleArn,
			RoleSessionName: aws.String("endly-e2e"),
		})
		if err != nil {
			return nil, err
		}
		awsCred = awsCred.WithCredentials(credentials.NewStaticCredentials(*result.Credentials.AccessKeyId, *result.Credentials.SecretAccessKey, *result.Credentials.SessionToken))
	}
	state := context.State()
	awsMap := data.NewMap()
	awsMap.Put("region", generic.Region)
	awsMap.Put("accountID", generic.Id)
	state.Put("aws", awsMap)
	_ = context.Put(configKey, awsCred)
	return awsCred, err
}



// GetClient get or creates aws client
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
