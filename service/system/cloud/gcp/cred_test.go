package gcp

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"google.golang.org/api/compute/v1"
	"log"
	"testing"
)

type testCtxClient struct {
	AbstractClient
	service *compute.Service
}

func (s *testCtxClient) SetService(service interface{}) error {
	var ok bool
	s.service, ok = service.(*compute.Service)
	if !ok {
		return fmt.Errorf("unable to set service: %T", service)
	}
	return nil
}
func (s *testCtxClient) Service() interface{} {
	return s.service
}

var testCtxServiceKey = (*testCtxClient)(nil)

func TestGetClient(t *testing.T) {

	if !HasTestCredentials() {
		return
	}
	manager := endly.New()
	context := manager.NewContext(nil)
	_, err := InitCredentials(context, map[string]interface{}{
		"Credentials": "gcp-e2e",
	})
	assert.Nil(t, err)

	var target = &testCtxClient{}
	err = GetClient(context, compute.New, testCtxServiceKey, target, compute.CloudPlatformScope)
	if !assert.Nil(t, err) {
		log.Print(err)
	}
	err = GetClient(context, compute.New, testCtxServiceKey, target, compute.CloudPlatformScope)
	if !assert.Nil(t, err) {
		log.Print(err)
	}

}
