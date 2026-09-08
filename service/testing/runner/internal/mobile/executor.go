package mobile

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/viant/assertly"
)

type Locator struct {
	Using string
	Value string
}

type LocatorResolver func(call Call) (Locator, bool, error)
type DeviceExecutor func(ctx context.Context, session *AppiumSession, call Call) (interface{}, error)

type ExecutionStep struct {
	Command    string
	Key        string
	Actual     interface{}
	Attempts   int
	DurationMs int
}

type ExecutionResult struct {
	Data        map[string]interface{}
	Steps       []ExecutionStep
	Validations []*assertly.Validation
}

type Executor struct {
	Session       *AppiumSession
	Resolve       LocatorResolver
	ExecuteDevice DeviceExecutor
	ActionTimeout time.Duration
	PollInterval  time.Duration
}

func (e *Executor) Run(ctx context.Context, commands []string) (*ExecutionResult, error) {
	if e.Session == nil || e.Resolve == nil {
		return nil, fmt.Errorf("mobile executor session and locator resolver are required")
	}
	if e.ActionTimeout <= 0 {
		e.ActionTimeout = 10 * time.Second
	}
	if e.PollInterval <= 0 {
		e.PollInterval = 100 * time.Millisecond
	}
	result := &ExecutionResult{Data: map[string]interface{}{}, Steps: []ExecutionStep{}, Validations: []*assertly.Validation{}}
	for _, source := range commands {
		parsed, err := ParseDSL(source)
		if err != nil {
			return result, fmt.Errorf("parse command %q: %w", source, err)
		}
		started := time.Now()
		actual, attempts, validation, err := e.execute(ctx, parsed, source)
		step := ExecutionStep{Command: source, Key: parsed.Key, Actual: actual, Attempts: attempts, DurationMs: int(time.Since(started) / time.Millisecond)}
		result.Steps = append(result.Steps, step)
		if validation != nil {
			result.Validations = append(result.Validations, validation)
		}
		if parsed.Key != "" {
			result.Data[parsed.Key] = actual
		}
		if err != nil {
			return result, fmt.Errorf("execute command %q: %w", source, err)
		}
	}
	return result, nil
}

func (e *Executor) execute(ctx context.Context, command *DSLCommand, source string) (interface{}, int, *assertly.Validation, error) {
	if command.Namespace == "device" {
		if command.Expectation {
			return nil, 0, nil, fmt.Errorf("device expectations are not implemented")
		}
		if e.ExecuteDevice == nil || len(command.Calls) != 1 {
			return nil, 0, nil, fmt.Errorf("unsupported device command")
		}
		actual, err := e.ExecuteDevice(ctx, e.Session, command.Calls[0])
		return actual, 1, nil, err
	}
	if len(command.Calls) != 2 {
		return nil, 0, nil, fmt.Errorf("app command requires one locator and one action")
	}
	locator, supported, err := e.Resolve(command.Calls[0])
	if err != nil {
		return nil, 0, nil, err
	}
	if !supported {
		return nil, 0, nil, fmt.Errorf("unsupported locator %q", command.Calls[0].Name)
	}
	action := command.Calls[1]
	timeout := e.ActionTimeout
	if command.Expectation {
		timeout = expectationTimeout(action, timeout)
	}
	deadlineCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	attempts := 0
	var actual interface{}
	var lastErr error
	for {
		attempts++
		actual, lastErr = e.tryElementAction(deadlineCtx, locator, action, command.Expectation)
		if lastErr == nil {
			if !command.Expectation {
				if strings.EqualFold(action.Name, "waitForVisible") {
					visible, _ := actual.(bool)
					if visible {
						return actual, attempts, nil, nil
					}
					lastErr = fmt.Errorf("element is not visible")
				} else {
					return actual, attempts, nil, nil
				}
			}
			if command.Expectation {
				matched, expected, matchErr := expectationMatches(action, actual)
				if matchErr != nil {
					return actual, attempts, nil, matchErr
				}
				if matched {
					validation := assertly.NewValidation()
					validation.Description = source
					validation.PassedCount = 1
					return actual, attempts, validation, nil
				}
				lastErr = fmt.Errorf("expectation did not match")
				_ = expected
			}
		}
		select {
		case <-deadlineCtx.Done():
			if command.Expectation {
				_, expected, matchErr := expectationMatches(action, actual)
				if matchErr != nil {
					return actual, attempts, nil, matchErr
				}
				validation := assertly.NewValidation()
				validation.Description = source
				failure := assertly.NewFailure("mobile", command.Calls[0].Name, assertly.EqualViolation, expected, actual)
				failure.Message = fmt.Sprintf("%s after %d attempt(s); last error: %v", failure.Message, attempts, lastErr)
				validation.AddFailure(failure)
				return actual, attempts, validation, nil
			}
			return actual, attempts, nil, fmt.Errorf("action timed out after %s: %w", timeout, lastErr)
		case <-time.After(e.PollInterval):
		}
	}
}

func (e *Executor) tryElementAction(ctx context.Context, locator Locator, action Call, expectation bool) (interface{}, error) {
	elementID, err := e.Session.FindElement(ctx, locator.Using, locator.Value)
	if err != nil {
		if expectation && strings.EqualFold(action.Name, "toBeVisible") {
			return false, nil
		}
		return nil, err
	}
	switch strings.ToLower(action.Name) {
	case "tap", "click":
		return nil, e.Session.Click(ctx, elementID)
	case "fill":
		value, err := requiredStringArg(action, 0)
		if err != nil {
			return nil, err
		}
		if err := e.Session.Clear(ctx, elementID); err != nil {
			return nil, err
		}
		return nil, e.Session.SendKeys(ctx, elementID, value)
	case "type":
		value, err := requiredStringArg(action, 0)
		if err != nil {
			return nil, err
		}
		return nil, e.Session.SendKeys(ctx, elementID, value)
	case "clear":
		return nil, e.Session.Clear(ctx, elementID)
	case "text", "tohavetext", "tocontaintext":
		return e.Session.Text(ctx, elementID)
	case "attribute", "tohaveattribute":
		name, err := requiredStringArg(action, 0)
		if err != nil {
			return nil, err
		}
		return e.Session.Attribute(ctx, elementID, name)
	case "displayed", "waitforvisible", "tobevisible":
		return e.Session.ElementBool(ctx, elementID, "displayed")
	case "enabled", "tobeenabled":
		return e.Session.ElementBool(ctx, elementID, "enabled")
	case "selected", "tobeselected":
		return e.Session.ElementBool(ctx, elementID, "selected")
	default:
		return nil, fmt.Errorf("unsupported element action %q", action.Name)
	}
}

func expectationMatches(call Call, actual interface{}) (bool, interface{}, error) {
	switch strings.ToLower(call.Name) {
	case "tobevisible", "tobeenabled", "tobeselected":
		value, ok := actual.(bool)
		return ok && value, true, nil
	case "tohavetext":
		expected, err := requiredStringArg(call, 0)
		return fmt.Sprint(actual) == expected, expected, err
	case "tocontaintext":
		expected, err := requiredStringArg(call, 0)
		return strings.Contains(fmt.Sprint(actual), expected), expected, err
	case "tohaveattribute":
		if len(call.Args) < 2 {
			return false, nil, fmt.Errorf("toHaveAttribute requires name and value")
		}
		return fmt.Sprint(actual) == fmt.Sprint(call.Args[1]), call.Args[1], nil
	default:
		return false, nil, fmt.Errorf("unsupported expectation %q", call.Name)
	}
}

func expectationTimeout(call Call, fallback time.Duration) time.Duration {
	index := -1
	switch strings.ToLower(call.Name) {
	case "tobevisible", "tobeenabled", "tobeselected":
		index = 0
	case "tohavetext", "tocontaintext":
		index = 1
	case "tohaveattribute":
		index = 2
	}
	if index < 0 || index >= len(call.Args) {
		return fallback
	}
	var milliseconds int64
	switch value := call.Args[index].(type) {
	case int64:
		milliseconds = value
	case float64:
		milliseconds = int64(value)
	case string:
		milliseconds, _ = strconv.ParseInt(value, 10, 64)
	}
	if milliseconds <= 0 {
		return fallback
	}
	return time.Duration(milliseconds) * time.Millisecond
}

func requiredStringArg(call Call, index int) (string, error) {
	if index >= len(call.Args) {
		return "", fmt.Errorf("%s requires argument %d", call.Name, index+1)
	}
	value, ok := call.Args[index].(string)
	if !ok {
		return "", fmt.Errorf("%s argument %d must be a string", call.Name, index+1)
	}
	return value, nil
}
