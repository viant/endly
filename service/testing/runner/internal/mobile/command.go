package mobile

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// Command is an argv-safe local command. Name and Args are never joined into a
// shell expression.
type Command struct {
	Name string
	Args []string
	Dir  string
	Env  map[string]string
}

type Result struct {
	Stdout     string
	Stderr     string
	ExitCode   int
	DurationMs int
}

type Process struct {
	PID  int
	Done <-chan error
	Stop func(ctx context.Context) error
}

// Runner is intentionally small so platform lifecycle code can be tested
// without an installed SDK or a real emulator.
type Runner interface {
	Run(ctx context.Context, command Command) (Result, error)
	Start(ctx context.Context, command Command, stdout, stderr io.Writer) (*Process, error)
}

type OSRunner struct{}

func (OSRunner) Run(ctx context.Context, command Command) (Result, error) {
	started := time.Now()
	cmd := exec.CommandContext(ctx, command.Name, command.Args...)
	cmd.Dir = command.Dir
	cmd.Env = commandEnv(command.Env)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := Result{
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		DurationMs: int(time.Since(started) / time.Millisecond),
	}
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}
	if err != nil {
		return result, fmt.Errorf("run %s: %w", command.Name, err)
	}
	return result, nil
}

func (OSRunner) Start(ctx context.Context, command Command, stdout, stderr io.Writer) (*Process, error) {
	// Started processes belong to their cleanup owner, not the transient action.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cmd := exec.Command(command.Name, command.Args...)
	cmd.Dir = command.Dir
	cmd.Env = commandEnv(command.Env)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	configureProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", command.Name, err)
	}
	done := make(chan error, 1)
	exited := make(chan struct{})
	go func() {
		done <- cmd.Wait()
		close(exited)
		close(done)
	}()
	stop := func(stopCtx context.Context) error {
		select {
		case <-exited:
			return nil
		default:
		}
		_ = interruptProcessGroup(cmd)
		select {
		case <-done:
			return nil
		case <-stopCtx.Done():
			if err := killProcessGroup(cmd); err != nil && !strings.Contains(strings.ToLower(err.Error()), "process already finished") {
				return err
			}
			return stopCtx.Err()
		}
	}
	return &Process{PID: cmd.Process.Pid, Done: done, Stop: stop}, nil
}

func commandEnv(values map[string]string) []string {
	if len(values) == 0 {
		return os.Environ()
	}
	env := append([]string(nil), os.Environ()...)
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		env = append(env, key+"="+values[key])
	}
	return env
}

func ResolveExecutable(name string, candidates ...string) (string, error) {
	if strings.ContainsRune(name, os.PathSeparator) {
		if isExecutable(name) {
			return name, nil
		}
		return "", fmt.Errorf("executable %q was not found", name)
	}
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}
	for _, candidate := range candidates {
		if isExecutable(candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("executable %q was not found", name)
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Mode()&0o111 != 0
}
