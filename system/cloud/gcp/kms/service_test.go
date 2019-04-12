package kms

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/gcp"
	"log"
	"testing"
)

func TestNew(t *testing.T) {

	if !gcp.HasTestCredentials() {
		return
	}

	context := endly.New().NewContext(nil)
	err := InitRequest(context, map[string]interface{}{
		"Credentials": "gcp-e2e",
	})
	assert.Nil(t, err)
	cred, _ := context.Secrets.GetCredentials("gcp-e2e")
	request, err := context.NewRequest(ServiceID, "subscriptionsList", map[string]interface{}{
		"project": fmt.Sprintf("projects/%v", cred.ProjectID),
	})
	assert.Nil(t, err)
	assert.NotNil(t, request)

}

func Test_Keys(t *testing.T) {
	context := endly.New().NewContext(nil)
	if !gcp.HasTestCredentials() {
		return
	}
	err := InitRequest(context, map[string]interface{}{
		"Credentials": "gcp-e2e",
	})
	assert.Nil(t, err)

	useCases := []struct {
		description string
		hasError    bool
		ring        string
		key         string
		purpose     string
		plainText   string
	}{
		{
			description: "symetric encryption",
			purpose:     "ENCRYPT_DECRYPT",
			ring:        "my_ring",
			key:         "my_key",
			plainText:   "this is secret",
		},
	}

	for _, useCase := range useCases {
		deployRequest := NewDeployKeyRequest("", useCase.ring, useCase.key, useCase.purpose)
		err = endly.Run(context, deployRequest, nil)
		if useCase.hasError {
			assert.NotNil(t, err, useCase.description)
			continue
		}
		if !assert.Nil(t, err, useCase.description) {
			log.Print(err)
			continue
		}
		encryptRequest := NewEncryptRequest("", useCase.ring, useCase.key, []byte(useCase.plainText))
		encryptResponse := &EncryptResponse{}
		err = endly.Run(context, encryptRequest, encryptResponse)
		if !assert.Nil(t, err, useCase.description) {
			log.Print(err)
			continue
		}
		decryptRequest := NewDecryptRequest("", useCase.ring, useCase.key, encryptResponse.CipherData)
		decryptResponse := &DecryptResponse{}
		err = endly.Run(context, decryptRequest, decryptResponse)
		if !assert.Nil(t, err, useCase.description) {
			log.Print(err)
			continue
		}
		assert.Equal(t, useCase.plainText, string(decryptResponse.PlainData), useCase.description)
	}

}
