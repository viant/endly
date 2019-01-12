package aws

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/secret"
	"os"
	"path"
	"testing"
)

func Test_GetAWSCredentialConfig(t *testing.T) {
	if ! toolbox.FileExists(path.Join(os.Getenv("HOME"), ".secret/aws.json")) {
		return
	}
	secretService := secret.New("", false)
	cred, err := secretService.GetCredentials("aws")
	if ! assert.Nil(t, err) {
		return
	}
	awsCred, err := GetAWSCredentialConfig(cred)
	if ! assert.Nil(t, err) {
		return
	}
	assert.NotNil(t, awsCred)
	assert.NotNil(t, cred.AccountID)
}
