package mobile

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type AppiumServerOptions struct {
	Mode             string
	ServerURL        string
	Executable       string
	AppiumHome       string
	Address          string
	Port             int
	BasePath         string
	LogPath          string
	StartupTimeoutMs int
}

func (o *AppiumServerOptions) Init() {
	if o.Mode == "" {
		o.Mode = "managed"
	}
	if o.Address == "" {
		o.Address = "127.0.0.1"
	}
	if o.Port == 0 {
		o.Port = 4723
	}
	if o.StartupTimeoutMs <= 0 {
		o.StartupTimeoutMs = 30_000
	}
	if o.Executable == "" {
		o.Executable = "appium"
	}
	if o.BasePath == "" {
		o.BasePath = "/"
	}
}

func (o AppiumServerOptions) Validate() error {
	switch o.Mode {
	case "managed":
		if o.Address != "127.0.0.1" && o.Address != "::1" && o.Address != "localhost" {
			return fmt.Errorf("managed Appium must bind to loopback, got %q", o.Address)
		}
		if o.Port < 1 || o.Port > 65535 {
			return fmt.Errorf("invalid Appium port %d", o.Port)
		}
	case "external":
		if o.ServerURL == "" {
			return fmt.Errorf("ServerURL is required for external Appium")
		}
	default:
		return fmt.Errorf("Mode must be managed or external")
	}
	return nil
}

type AppiumServer struct {
	ID        string
	Endpoint  string
	Ownership string
	PID       int
	LogPath   string

	process *Process
	log     io.Closer
	stop    sync.Once
	stopErr error
}

func StartAppium(ctx context.Context, runner Runner, options AppiumServerOptions) (*AppiumServer, error) {
	options.Init()
	if err := options.Validate(); err != nil {
		return nil, err
	}
	if options.Mode == "external" {
		client, err := NewAppiumClient(options.ServerURL, nil)
		if err != nil {
			return nil, err
		}
		if _, err := client.Status(ctx); err != nil {
			return nil, fmt.Errorf("external Appium health check: %w", err)
		}
		return &AppiumServer{ID: "external:" + options.ServerURL, Endpoint: options.ServerURL, Ownership: "external"}, nil
	}
	executable, err := ResolveExecutable(options.Executable)
	if err != nil {
		return nil, err
	}
	stdout, stderr, closer, err := appiumLog(options.LogPath)
	if err != nil {
		return nil, err
	}
	args := []string{
		"--address", options.Address,
		"--port", strconv.Itoa(options.Port),
		"--base-path", options.BasePath,
		"--log-no-colors",
	}
	env := map[string]string{}
	if options.AppiumHome != "" {
		env["APPIUM_HOME"] = options.AppiumHome
	}
	process, err := runner.Start(ctx, Command{Name: executable, Args: args, Env: env}, stdout, stderr)
	if err != nil {
		_ = closer.Close()
		return nil, err
	}
	endpoint := "http://" + options.Address + ":" + strconv.Itoa(options.Port) + strings.TrimRight(options.BasePath, "/")
	server := &AppiumServer{
		ID:        fmt.Sprintf("managed:%d:%d", process.PID, time.Now().UnixNano()),
		Endpoint:  endpoint,
		Ownership: "managed",
		PID:       process.PID,
		LogPath:   options.LogPath,
		process:   process,
		log:       closer,
	}
	client, err := NewAppiumClient(endpoint, nil)
	if err != nil {
		_ = server.Stop(context.Background())
		return nil, err
	}
	healthCtx, cancel := context.WithTimeout(ctx, time.Duration(options.StartupTimeoutMs)*time.Millisecond)
	defer cancel()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	var lastErr error
	for {
		if _, lastErr = client.Status(healthCtx); lastErr == nil {
			return server, nil
		}
		select {
		case processErr, open := <-process.Done:
			if open && processErr != nil {
				lastErr = processErr
			}
			_ = server.Stop(context.Background())
			return nil, fmt.Errorf("Appium exited before becoming ready: %w", lastErr)
		case <-healthCtx.Done():
			_ = server.Stop(context.Background())
			return nil, fmt.Errorf("wait for Appium readiness: %w (last health error: %v)", healthCtx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

func (s *AppiumServer) Stop(ctx context.Context) error {
	if s == nil || s.Ownership != "managed" {
		return nil
	}
	s.stop.Do(func() {
		if s.process != nil && s.process.Stop != nil {
			s.stopErr = s.process.Stop(ctx)
		}
		if s.log != nil {
			if err := s.log.Close(); s.stopErr == nil {
				s.stopErr = err
			}
		}
	})
	return s.stopErr
}

func appiumLog(path string) (io.Writer, io.Writer, io.Closer, error) {
	if path == "" {
		return io.Discard, io.Discard, io.NopCloser(strings.NewReader("")), nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, nil, nil, fmt.Errorf("create Appium log directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open Appium log: %w", err)
	}
	return file, file, file, nil
}
