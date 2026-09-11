package android

import (
	"context"
	"github.com/viant/endly"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCloseRetriesFailedBackendDeletion(t *testing.T) {
	attempts := 0
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodDelete {
			attempts++
			if attempts == 1 {
				w.WriteHeader(500)
				w.Write([]byte(`{"value":{"error":"unknown error","message":"temporary"}}`))
				return
			}
		}
		w.Write([]byte(`{"value":{}}`))
	}))
	defer backend.Close()
	svc := newService(&fakeRunner{})
	ctx := endly.New().NewContext(nil)
	attached, err := svc.attach(ctx, &AttachRequest{SessionID: "retry", BackendSessionID: "backend", ServerURL: backend.URL, TakeOwnership: true, TestIDStrategy: "accessibilityId"})
	if err != nil {
		t.Fatal(err)
	}
	request := &CloseRequest{SessionID: attached.Session.ID}
	if _, err := svc.close(context.Background(), request); err == nil {
		t.Fatal("expected first DELETE to fail")
	}
	if _, exists := svc.sessions[request.SessionID]; !exists {
		t.Fatal("lost retry handle")
	}
	if _, err := svc.close(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if attempts != 2 {
		t.Fatalf("DELETE attempts=%d", attempts)
	}
}
