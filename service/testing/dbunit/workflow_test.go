package dbunit_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	manager "github.com/viant/endly/service/manager"
	_ "github.com/viant/endly/service/testing/dbunit"
	_ "github.com/viant/endly/service/testing/runner/http"
)

func TestWorkflowCommittedFixturesVisibleToHTTPApplication(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.db")
	application, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	defer application.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result, err := application.ExecContext(r.Context(), "UPDATE users SET name = 'updated' WHERE name = 'before'")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		rows, _ := result.RowsAffected()
		if rows != 1 {
			http.Error(w, "fixture not committed", http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	runtime := manager.New(endly.New)
	defer runtime.Shutdown(context.Background())
	session, err := runtime.Open(context.Background(), &manager.OpenRequest{})
	require.NoError(t, err)
	content := fmt.Sprintf(`pipeline:
  connection:
    action: dbunit:register
    database: app
    driver: sqlite
    dsn: %s
  schema:
    action: dbunit:script
    database: app
    statements:
      - CREATE TABLE users(id INTEGER PRIMARY KEY, name TEXT)
  prepare:
    action: dbunit:prepare
    database: app
    datasets:
      - table: [users]
        source:
          kind: maps
          data:
            - id: 1
              name: $initialName
  application:
    action: http/runner:send
    requests:
      - method: POST
        url: %s
        expect:
          Code: 204
  expect:
    action: dbunit:expect
    database: app
    expectations:
      - table: [users]
        expected:
          kind: maps
          data:
            - id: 1
              name: updated
`, path, server.URL)
	_, err = runtime.LoadWorkflow(context.Background(), &manager.LoadWorkflowRequest{SessionID: session.SessionID, URL: filepath.Join(t.TempDir(), "run.yaml"), Content: content, Format: "yaml"})
	require.NoError(t, err)
	operation, err := runtime.StartWorkflow(&manager.RunWorkflowRequest{SessionID: session.SessionID, Workflow: "run", Tasks: "*", Params: map[string]any{"initialName": "before"}})
	require.NoError(t, err)
	done, err := runtime.WaitOperation(context.Background(), session.SessionID, operation.ID)
	require.NoError(t, err)
	require.Equal(t, manager.OperationSucceeded, done.Status, done.Error)
	require.Zero(t, done.FailedAssertions)
	require.Positive(t, done.PassedAssertions)
}
