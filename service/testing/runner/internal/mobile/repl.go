package mobile

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
)

type REPLConfig struct {
	Prompt         string
	MaxSourceBytes int
	MaxTreeNodes   int
	FailOnError    bool
	HistoryPath    string
	MaxHistory     int
}

type REPLCallbacks struct {
	Execute    func(context.Context, string) (*ExecutionResult, error)
	Source     func(context.Context) (string, error)
	Screenshot func(context.Context) (*Evidence, error)
	Status     func(context.Context) interface{}
	Close      func(context.Context) error
}

type REPLResult struct {
	Commands  int
	Errors    int
	ExitedBy  string
	History   []string
	Data      map[string]interface{}
	Artifacts []*Evidence
}

type replInput struct {
	line string
	err  error
	eof  bool
}

func RunREPL(ctx context.Context, input io.Reader, output io.Writer, config REPLConfig, callbacks REPLCallbacks) (*REPLResult, error) {
	if input == nil || output == nil || callbacks.Execute == nil || callbacks.Source == nil {
		return nil, fmt.Errorf("REPL input, output, execute, and source callbacks are required")
	}
	if config.Prompt == "" {
		config.Prompt = "mobile> "
	}
	if config.MaxSourceBytes <= 0 {
		config.MaxSourceBytes = 2_000_000
	}
	if config.MaxTreeNodes <= 0 {
		config.MaxTreeNodes = 500
	}
	history, err := LoadHistory(config.HistoryPath, config.MaxHistory)
	if err != nil {
		return nil, err
	}
	result := &REPLResult{History: history, Data: map[string]interface{}{}, Artifacts: []*Evidence{}}
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), 1_000_000)
	replCtx, stopSignals := signal.NotifyContext(ctx, os.Interrupt)
	defer stopSignals()
	inputs := make(chan replInput)
	go func() {
		for scanner.Scan() {
			select {
			case inputs <- replInput{line: scanner.Text()}:
			case <-replCtx.Done():
				return
			}
		}
		entry := replInput{eof: true, err: scanner.Err()}
		select {
		case inputs <- entry:
		case <-replCtx.Done():
		}
		close(inputs)
	}()
	writer := bufio.NewWriter(output)
	defer writer.Flush()
	writeREPLHelp(writer)
	for {
		_, _ = fmt.Fprint(writer, config.Prompt)
		_ = writer.Flush()
		var inputEntry replInput
		select {
		case <-replCtx.Done():
			if ctx.Err() != nil {
				result.ExitedBy = "context"
				return result, ctx.Err()
			}
			result.ExitedBy = "interrupt"
			_, _ = fmt.Fprintln(writer, "\ninterrupted")
			return result, nil
		case entry, ok := <-inputs:
			if !ok || entry.eof {
				result.ExitedBy = "eof"
				if entry.err != nil {
					return result, fmt.Errorf("read REPL input: %w", entry.err)
				}
				return result, nil
			}
			inputEntry = entry
		}
		line := strings.TrimSpace(inputEntry.line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "!") {
			replayed, err := replayHistory(line, result.History)
			if err != nil {
				if stopErr := replError(writer, result, config, err); stopErr != nil {
					return result, stopErr
				}
				continue
			}
			line = replayed
			_, _ = fmt.Fprintf(writer, "%s\n", line)
		}
		result.History = append(result.History, line)
		if !strings.EqualFold(line, ":clear-history") {
			if err := AppendHistory(config.HistoryPath, line); err != nil {
				if stopErr := replError(writer, result, config, err); stopErr != nil {
					return result, stopErr
				}
			}
		}
		if strings.HasPrefix(line, ":") {
			exit, err := runMetaCommand(ctx, writer, line, config, callbacks, result)
			if err != nil {
				if stopErr := replError(writer, result, config, err); stopErr != nil {
					return result, stopErr
				}
				continue
			}
			if exit != "" {
				result.ExitedBy = exit
				return result, nil
			}
			continue
		}
		execution, err := callbacks.Execute(ctx, line)
		result.Commands++
		if err != nil {
			if stopErr := replError(writer, result, config, err); stopErr != nil {
				return result, stopErr
			}
			continue
		}
		for key, value := range execution.Data {
			result.Data[key] = value
		}
		for _, line := range SortedDataLines(execution.Data) {
			_, _ = fmt.Fprintln(writer, line)
		}
		for _, step := range execution.Steps {
			_, _ = fmt.Fprintf(writer, "ok (%dms, %d attempt(s))\n", step.DurationMs, step.Attempts)
		}
		for _, validation := range execution.Validations {
			if validation.HasFailure() {
				result.Errors++
				_, _ = fmt.Fprintf(writer, "FAIL: %s\n", validation.Report())
				if config.FailOnError {
					return result, fmt.Errorf("REPL assertion failed: %s", validation.Report())
				}
			} else {
				_, _ = fmt.Fprintf(writer, "PASS: %s\n", validation.Description)
			}
		}
		for _, failure := range execution.Failures {
			for _, artifact := range failure.Artifacts {
				result.Artifacts = append(result.Artifacts, artifact)
				_, _ = fmt.Fprintf(writer, "failure evidence: %s (%s)\n", artifact.URL, artifact.Kind)
			}
			for _, captureErr := range failure.Errors {
				_, _ = fmt.Fprintf(writer, "failure evidence error: %s\n", captureErr)
			}
		}
	}
}

func runMetaCommand(ctx context.Context, writer io.Writer, line string, config REPLConfig, callbacks REPLCallbacks, result *REPLResult) (string, error) {
	command, argument := splitMetaCommand(line)
	switch command {
	case ":help", ":h", ":?":
		writeREPLHelp(writer)
	case ":quit", ":exit", ":q":
		return "quit", nil
	case ":close":
		if callbacks.Close == nil {
			return "", fmt.Errorf("close is not available")
		}
		if err := callbacks.Close(ctx); err != nil {
			return "", err
		}
		_, _ = fmt.Fprintln(writer, "session closed")
		return "close", nil
	case ":status":
		if callbacks.Status == nil {
			return "", fmt.Errorf("status is not available")
		}
		data, err := json.MarshalIndent(callbacks.Status(ctx), "", "  ")
		if err != nil {
			return "", err
		}
		_, _ = fmt.Fprintln(writer, string(data))
	case ":source":
		source, err := callbacks.Source(ctx)
		if err != nil {
			return "", err
		}
		if len(source) > config.MaxSourceBytes {
			source = source[:config.MaxSourceBytes] + "\n... source truncated"
		}
		_, _ = fmt.Fprintln(writer, source)
	case ":tree", ":find":
		source, err := callbacks.Source(ctx)
		if err != nil {
			return "", err
		}
		tree, err := SummarizeHierarchy(source, argument, config.MaxTreeNodes)
		if err != nil {
			return "", err
		}
		if tree == "" {
			tree = "no matching nodes"
		}
		_, _ = fmt.Fprintln(writer, tree)
	case ":screenshot":
		if callbacks.Screenshot == nil {
			return "", fmt.Errorf("screenshot is not available")
		}
		artifact, err := callbacks.Screenshot(ctx)
		if err != nil {
			return "", err
		}
		result.Artifacts = append(result.Artifacts, artifact)
		_, _ = fmt.Fprintf(writer, "%s (%d bytes)\n", artifact.URL, artifact.Size)
	case ":history":
		for index, entry := range result.History[:len(result.History)-1] {
			_, _ = fmt.Fprintf(writer, "%d  %s\n", index+1, entry)
		}
	case ":clear-history":
		if err := ClearHistory(config.HistoryPath); err != nil {
			return "", err
		}
		result.History = []string{}
		_, _ = fmt.Fprintln(writer, "history cleared")
	default:
		return "", fmt.Errorf("unknown REPL command %q; use :help", command)
	}
	return "", nil
}

func writeREPLHelp(writer io.Writer) {
	_, _ = fmt.Fprintln(writer, "Mobile REPL: enter one app.* or device.* DSL command per line.")
	_, _ = fmt.Fprintln(writer, "  :status  :source  :tree [filter]  :find <text>  :screenshot")
	_, _ = fmt.Fprintln(writer, "  :history  :clear-history  !<number>  :help  :close  :quit")
}

func splitMetaCommand(line string) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(line), " ", 2)
	if len(parts) == 1 {
		return strings.ToLower(parts[0]), ""
	}
	return strings.ToLower(parts[0]), strings.TrimSpace(parts[1])
}

func replayHistory(line string, history []string) (string, error) {
	index, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "!")))
	if err != nil || index <= 0 || index > len(history) {
		return "", fmt.Errorf("history entry %q does not exist", line)
	}
	return history[index-1], nil
}

func replError(writer io.Writer, result *REPLResult, config REPLConfig, err error) error {
	result.Errors++
	_, _ = fmt.Fprintf(writer, "error: %v\n", err)
	if config.FailOnError {
		return err
	}
	return nil
}
