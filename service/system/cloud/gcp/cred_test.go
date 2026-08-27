package gcp

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	"google.golang.org/api/compute/v1"
	"log"
	"os"
	"path/filepath"
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

func TestInitCredentials_passThroughWhenAliasNotInMapping(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	secretDir := filepath.Join(home, ".secret")
	require.NoError(t, os.MkdirAll(secretDir, 0o700))
	t.Setenv("HOME", home)
	require.NoError(t, os.WriteFile(filepath.Join(secretDir, "gcp-e2e.json"), []byte(`{
  "type": "service_account",
  "project_id": "gcp-e2e",
  "private_key_id": "abc",
  "private_key": "-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----\n",
  "client_email": "test@gcp-e2e.iam.gserviceaccount.com"
}`), 0o600))

	mapping := filepath.Join(dir, "e2e-credentials.yaml")
	require.NoError(t, os.WriteFile(mapping, []byte(`credentials:
  other: file:///tmp/x.json
`), 0o600))
	t.Setenv("E2E_CREDENTIALS_FILE", mapping)

	manager := endly.New()
	context := manager.NewContext(nil)
	cfg, err := InitCredentials(context, map[string]interface{}{
		"Credentials": "gcp-e2e",
	})
	assert.NoError(t, err)
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.Secret)
}

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
