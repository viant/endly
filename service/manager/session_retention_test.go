package manager

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
)

func TestCleanupRetainsActiveSession(t *testing.T) {
	for _, status := range []string{OperationQueued, OperationRunning, OperationCancelling} {
		t.Run(status, func(t *testing.T) {
			service := New(endly.New, WithSessionTTL(time.Minute))
			defer service.Shutdown(context.Background())
			info, err := service.Open(context.Background(), &OpenRequest{})
			require.NoError(t, err)
			session, err := service.lookup(info.SessionID)
			require.NoError(t, err)
			now := time.Now().UTC()
			session.mu.Lock()
			session.lastAccess = now.Add(-time.Hour)
			session.operations["active"] = &Operation{ID: "active", Status: status}
			session.mu.Unlock()
			service.cleanup(now)
			operation, err := service.GetOperation(info.SessionID, "active")
			require.NoError(t, err, "idle client must not expire an active execution")
			require.Equal(t, status, operation.Status)
		})
	}
}

func TestCompletionRenewsSessionIdleTTL(t *testing.T) {
	for _, kind := range []string{"action", "workflow"} {
		t.Run(kind, func(t *testing.T) {
			service := New(endly.New, WithSessionTTL(time.Minute))
			defer service.Shutdown(context.Background())
			info, err := service.Open(context.Background(), &OpenRequest{})
			require.NoError(t, err)
			_, err = service.LoadWorkflow(context.Background(), &LoadWorkflowRequest{SessionID: info.SessionID, URL: "retention.yaml", Format: "yaml", Content: "pipeline:\n  work:\n    action: nop\n"})
			require.NoError(t, err)
			session, err := service.lookup(info.SessionID)
			require.NoError(t, err)
			session.execMu.Lock()
			var operation *Operation
			if kind == "action" {
				operation, err = service.StartAction(&RunActionRequest{SessionID: info.SessionID, Service: "workflow", Action: "nop"})
			} else {
				operation, err = service.StartWorkflow(&RunWorkflowRequest{SessionID: info.SessionID, Workflow: "retention"})
			}
			session.mu.Lock()
			session.lastAccess = time.Now().UTC().Add(-time.Hour)
			session.mu.Unlock()
			session.execMu.Unlock()
			require.NoError(t, err)
			// Observe completion without a control lookup renewing lastAccess.
			require.Eventually(t, func() bool {
				session.mu.RLock()
				defer session.mu.RUnlock()
				return session.operations[operation.ID].FinishedAt != nil
			}, time.Second, time.Millisecond)
			service.cleanup(time.Now().UTC())
			finished, err := service.GetOperation(info.SessionID, operation.ID)
			require.NoError(t, err, "completed result must remain readable for the idle TTL")
			require.Equal(t, OperationSucceeded, finished.Status)
			service.cleanup(time.Now().UTC().Add(2 * time.Minute))
			_, err = service.GetOperation(info.SessionID, operation.ID)
			require.ErrorContains(t, err, "was not found", "idle completed sessions still expire")
		})
	}
}
