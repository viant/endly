package mobile

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

type serverRunner struct {
	stopped atomic.Bool
}

func (s *serverRunner) Run(context.Context, Command) (Result, error) { return Result{}, nil }

func (s *serverRunner) Start(context.Context, Command, io.Writer, io.Writer) (*Process, error) {
	done := make(chan error)
	return &Process{
		PID:  1234,
		Done: done,
		Stop: func(context.Context) error {
			s.stopped.Store(true)
			close(done)
			return nil
		},
	}, nil
}

func TestExternalAppiumServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"value": map[string]interface{}{"ready": true}})
	}))
	defer server.Close()
	handle, err := StartAppium(context.Background(), &serverRunner{}, AppiumServerOptions{Mode: "external", ServerURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if handle.Ownership != "external" || handle.Endpoint != server.URL {
		t.Fatalf("unexpected external server: %+v", handle)
	}
}

func TestManagedAppiumServerLifecycle(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	httpServer := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"value": map[string]interface{}{"ready": true}})
	})}
	go func() { _ = httpServer.Serve(listener) }()
	defer httpServer.Close()

	executable := filepath.Join(t.TempDir(), "appium")
	if err := os.WriteFile(executable, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	runner := &serverRunner{}
	handle, err := StartAppium(context.Background(), runner, AppiumServerOptions{
		Mode:       "managed",
		Executable: executable,
		Address:    "127.0.0.1",
		Port:       port,
	})
	if err != nil {
		t.Fatal(err)
	}
	if handle.Ownership != "managed" || handle.PID != 1234 {
		t.Fatalf("unexpected managed server: %+v", handle)
	}
	if err := handle.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !runner.stopped.Load() {
		t.Fatal("expected managed process to stop")
	}
	if err := handle.Stop(context.Background()); err != nil {
		t.Fatalf("idempotent stop failed: %v", err)
	}
}

func TestManagedAppiumRejectsPublicBind(t *testing.T) {
	options := AppiumServerOptions{Mode: "managed", Address: "0.0.0.0", Port: 4723}
	options.Init()
	if err := options.Validate(); err == nil {
		t.Fatal("expected public bind validation failure")
	}
}
