package client

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	"github.com/viant/endly/server/control"
	managerservice "github.com/viant/endly/service/manager"
	"github.com/viant/scy"
	_ "github.com/viant/scy/kms/blowfish"
)

type handlerTransport struct {
	handler http.Handler
}

func (h *handlerTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	recorder := httptest.NewRecorder()
	h.handler.ServeHTTP(recorder, request)
	return recorder.Result(), nil
}

func TestClientLifecycle(t *testing.T) {
	runtime := managerservice.New(endly.New)
	output := &bytes.Buffer{}
	cli := &Client{
		Endpoint: "http://endly.test",
		HTTP:     &http.Client{Transport: &handlerTransport{handler: control.New(runtime)}},
		Output:   output,
	}
	require.NoError(t, cli.open(nil))
	require.NotEmpty(t, cli.SessionID)

	workflowPath := filepath.Join(t.TempDir(), "sample.yaml")
	require.NoError(t, os.WriteFile(workflowPath, []byte(`pipeline:
  first:
    action: nop
`), 0o600))
	require.NoError(t, cli.load([]string{workflowPath, "--alias", "sample"}))
	require.NoError(t, cli.runWorkflow([]string{"sample", "--tasks", "*"}))
	require.Contains(t, output.String(), "workflow.nop")
	require.Contains(t, output.String(), "Operation ")
	require.NoError(t, cli.inspect([]string{"--full"}))
	require.NoError(t, cli.logging([]string{"disable"}))
	require.NoError(t, cli.unload([]string{"sample"}))
	require.NoError(t, cli.close())
}

func TestClientProfilePersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client.json")
	store := &ProfileStore{Profiles: map[string]*Profile{}}
	cli := &Client{Endpoint: "http://127.0.0.1:8080", SessionID: "session-1", ProfileName: "dev", Profiles: store, ProfilePath: path}
	require.NoError(t, cli.saveProfile())
	loaded, err := loadProfiles(path)
	require.NoError(t, err)
	require.Equal(t, "session-1", loaded.Profiles["dev"].SessionID)
	require.Equal(t, "http://127.0.0.1:8080", loaded.Profiles["dev"].Endpoint)
}

func TestClientScyJWTAuthentication(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	privatePEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	publicDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER})
	privatePath := filepath.Join(t.TempDir(), "private.scy")
	publicPath := filepath.Join(t.TempDir(), "public.scy")
	key := "blowfish://default"
	scyService := scy.New()
	require.NoError(t, scyService.Store(context.Background(), scy.NewSecret(string(privatePEM), &scy.Resource{URL: privatePath, Key: key})))
	require.NoError(t, scyService.Store(context.Background(), scy.NewSecret(string(publicPEM), &scy.Resource{URL: publicPath, Key: key})))
	authenticator, err := control.NewJWTAuthenticator(context.Background(), publicPath+"|"+key, "endly-client", "endly-service", "endly:execute")
	require.NoError(t, err)
	provider, err := NewJWTTokenProvider(context.Background(), privatePath+"|"+key, "endly-client", "endly-service", "endly:execute", "test-user", time.Minute)
	require.NoError(t, err)
	runtime := managerservice.New(endly.New)
	output := &bytes.Buffer{}
	cli := &Client{
		Endpoint:      "http://endly.test",
		TokenProvider: provider,
		HTTP:          &http.Client{Transport: &handlerTransport{handler: control.NewAuthenticated(runtime, authenticator)}},
		Output:        output,
	}
	require.NoError(t, cli.open([]string{"--no-save"}))
	info, err := runtime.Session(cli.SessionID)
	require.NoError(t, err)
	require.Equal(t, "test-user", info.Subject)

	wrongAudience, err := NewJWTTokenProvider(context.Background(), privatePath+"|"+key, "endly-client", "wrong-service", "endly:execute", "test-user", time.Minute)
	require.NoError(t, err)
	badToken, err := wrongAudience(context.Background())
	require.NoError(t, err)
	_, err = authenticator.Authenticate(context.Background(), badToken)
	require.ErrorContains(t, err, "audience")

	expiring, err := NewJWTTokenProvider(context.Background(), privatePath+"|"+key, "endly-client", "endly-service", "endly:execute", "test-user", time.Millisecond)
	require.NoError(t, err)
	expiredToken, err := expiring(context.Background())
	require.NoError(t, err)
	time.Sleep(5 * time.Millisecond)
	_, err = authenticator.Authenticate(context.Background(), expiredToken)
	require.Error(t, err)
}
