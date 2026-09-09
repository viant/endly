package mobile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const SessionDescriptorVersion = 1

// SessionDescriptor is the portable, non-secret information needed by another
// Endly process to reconnect to an existing Appium session.
type SessionDescriptor struct {
	Version          int       `json:"version"`
	Platform         string    `json:"platform"`
	SessionID        string    `json:"sessionId"`
	BackendSessionID string    `json:"backendSessionId"`
	Endpoint         string    `json:"endpoint"`
	TargetID         string    `json:"targetId,omitempty"`
	TestIDStrategy   string    `json:"testIdStrategy,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
}

func (d *SessionDescriptor) Init() {
	if d.Version == 0 {
		d.Version = SessionDescriptorVersion
	}
	if d.CreatedAt.IsZero() {
		d.CreatedAt = time.Now().UTC()
	}
}

func (d *SessionDescriptor) Validate(expectedPlatform string) error {
	if d == nil {
		return fmt.Errorf("session descriptor is required")
	}
	if d.Version != SessionDescriptorVersion {
		return fmt.Errorf("unsupported session descriptor version %d", d.Version)
	}
	if d.Platform == "" || d.BackendSessionID == "" || d.Endpoint == "" {
		return fmt.Errorf("session descriptor Platform, BackendSessionID, and Endpoint are required")
	}
	if expectedPlatform != "" && !strings.EqualFold(d.Platform, expectedPlatform) {
		return fmt.Errorf("session descriptor platform %q does not match %q", d.Platform, expectedPlatform)
	}
	return nil
}

func WriteSessionDescriptor(path string, descriptor *SessionDescriptor) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("session descriptor path is required")
	}
	descriptor.Init()
	if err := descriptor.Validate(""); err != nil {
		return err
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create session descriptor directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".endly-session-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary session descriptor: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(descriptor); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("encode session descriptor: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync session descriptor: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close session descriptor: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("publish session descriptor: %w", err)
	}
	return nil
}

func ReadSessionDescriptor(path, expectedPlatform string) (*SessionDescriptor, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read session descriptor: %w", err)
	}
	descriptor := &SessionDescriptor{}
	if err := json.Unmarshal(data, descriptor); err != nil {
		return nil, fmt.Errorf("decode session descriptor: %w", err)
	}
	if err := descriptor.Validate(expectedPlatform); err != nil {
		return nil, err
	}
	return descriptor, nil
}

func RemoveSessionDescriptor(path string) error {
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove session descriptor: %w", err)
	}
	return nil
}
