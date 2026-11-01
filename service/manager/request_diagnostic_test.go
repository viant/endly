package manager

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	"github.com/viant/endly/service/credential"
	_ "github.com/viant/endly/service/testing/runner/webdriver"
	"github.com/viant/scy/cred"
)

type diagnosticSecrets struct {
	credential.SecretLookup
	password string
}

func (s diagnosticSecrets) GetCredentials(context.Context, string) (*cred.Generic, error) {
	result := &cred.Generic{}
	result.Password = s.password
	return result, nil
}

func TestWorkflowConversionDiagnosticsRedactSecretWithLoggingDisabled(t *testing.T) {
	const sentinel = "SYNTHETIC_PASSWORD_MANAGER_51ab0c"
	service := New(endly.New)
	session, err := service.Open(context.Background(), &OpenRequest{})
	require.NoError(t, err)
	defer service.Close(context.Background(), session.SessionID)
	service.sessions[session.SessionID].context.Secrets = diagnosticSecrets{password: sentinel}
	_, err = service.SetLogging(&SetLoggingRequest{SessionID: session.SessionID, Enabled: false})
	require.NoError(t, err)
	_, err = service.LoadWorkflow(context.Background(), &LoadWorkflowRequest{
		SessionID: session.SessionID, URL: "synthetic-conversion.yaml", Format: "yaml", Content: `init:
  login: ${secrets.synthetic}
pipeline:
  convert:
    action: webdriver:run
    actions:
      - calls:
          - method: SendKeys
            parameters:
              - $login.Password
          - method: Click
            parameters:
              invalid: []
`,
	})
	require.NoError(t, err)
	op, err := service.StartWorkflow(&RunWorkflowRequest{SessionID: session.SessionID, Workflow: "synthetic-conversion", Tasks: "*"})
	require.NoError(t, err)
	op, err = service.WaitOperation(context.Background(), session.SessionID, op.ID)
	require.NoError(t, err)
	require.Equal(t, OperationFailed, op.Status)
	require.Contains(t, op.Error, "*webdriver.RunRequest")
	require.Contains(t, op.Error, "map[string]interface {} into type *[]interface {}")
	// Exercise the same public native route used by MCP getOperation, including
	// events and diagnostics as well as the top-level operation error.
	response, err := service.GetOperation(session.SessionID, op.ID)
	require.NoError(t, err)
	encoded, err := json.Marshal(response)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), sentinel)
	require.Contains(t, string(encoded), "[REDACTED]")
}
