package mobile

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type LoggedProcess struct {
	PID     int
	Path    string
	process *Process
	file    *os.File
	stop    sync.Once
	stopErr error
}

func StartLoggedProcess(ctx context.Context, runner Runner, command Command, logPath string) (*LoggedProcess, error) {
	if logPath == "" {
		return nil, fmt.Errorf("log path is required")
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return nil, fmt.Errorf("create capture directory: %w", err)
	}
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open capture log: %w", err)
	}
	process, err := runner.Start(ctx, command, file, file)
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	return &LoggedProcess{PID: process.PID, Path: logPath, process: process, file: file}, nil
}

func (p *LoggedProcess) Stop(ctx context.Context) error {
	if p == nil {
		return nil
	}
	p.stop.Do(func() {
		stopCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if p.process != nil && p.process.Stop != nil {
			p.stopErr = p.process.Stop(stopCtx)
		}
		if p.file != nil {
			if err := p.file.Close(); p.stopErr == nil {
				p.stopErr = err
			}
		}
	})
	return p.stopErr
}
