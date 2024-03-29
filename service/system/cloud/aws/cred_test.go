package aws

import (
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"

	"github.com/viant/toolbox"
	"github.com/viant/scy/cred/secret"
	"os"
	"path"
	"testing"
)

func Test_GetAWSCredentialConfig(t *testing.T) {
	if !toolbox.FileExists(path.Join(os.Getenv("HOME"), ".secret/aws.json")) {
		return
	}
	secretService := secret.New("", false)
	cred, err := secretService.GetCredentials("aws")
	if !assert.Nil(t, err) {
		return
	}
	awsCred, err := GetAWSCredentialConfig(cred)
	if !assert.Nil(t, err) {
		return
	}
	assert.NotNil(t, awsCred)
	assert.NotNil(t, cred.AccountID)
}

func Test_GetClient(t *testing.T) {
	if !toolbox.FileExists(path.Join(os.Getenv("HOME"), ".secret/aws.json")) {
		return
	}
	manager := endly.New()
	context := manager.NewContext(nil)
	var key = struct{}{}
	_, err := InitCredentials(context, map[string]interface{}{
		"Credentials": "aws",
	}, key)
	assert.Nil(t, err)
	client := &iam.IAM{}
	err = GetClient(context, iam.New, &client)
	assert.Nil(t, err)
	output, err := client.ListUsers(&iam.ListUsersInput{})
	assert.Nil(t, err)
	assert.True(t, len(output.Users) > 0)
}
