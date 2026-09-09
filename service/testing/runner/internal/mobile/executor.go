package mobile

import (
	"context"
	"fmt"
	"math"
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
	Failures    []*FailureEvidence
}

type Executor struct {
	Session       *AppiumSession
	Resolve       LocatorResolver
	ExecuteDevice DeviceExecutor
	ActionTimeout time.Duration
	PollInterval  time.Duration
}

type elementTarget struct {
	locator   Locator
	selection string // strict, first, last, nth
	index     int
}

func (e *Executor) Run(ctx context.Context, commands []string) (*ExecutionResult, error) {
	candidates := make([]interface{}, len(commands))
	for index, command := range commands {
		candidates[index] = command
	}
	return e.RunAny(ctx, candidates)
}

func (e *Executor) RunAny(ctx context.Context, commands []interface{}) (*ExecutionResult, error) {
	if e.Session == nil || e.Resolve == nil {
		return nil, fmt.Errorf("mobile executor session and locator resolver are required")
	}
	if e.ActionTimeout <= 0 {
		e.ActionTimeout = 10 * time.Second
	}
	if e.PollInterval <= 0 {
		e.PollInterval = 100 * time.Millisecond
	}
	result := &ExecutionResult{Data: map[string]interface{}{}, Steps: []ExecutionStep{}, Validations: []*assertly.Validation{}, Failures: []*FailureEvidence{}}
	for _, candidate := range commands {
		parsed, source, err := ParseInstruction(candidate)
		if err != nil {
			return result, fmt.Errorf("parse command %q: %w", source, err)
		}
		started := time.Now()
		actual, attempts, validation, err := e.execute(ctx, parsed, source)
		result.Steps = append(result.Steps, ExecutionStep{
			Command: source, Key: parsed.Key, Actual: actual, Attempts: attempts,
			DurationMs: int(time.Since(started) / time.Millisecond),
		})
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
		return e.executeDevice(ctx, command, source)
	}
	if len(command.Calls) < 2 || len(command.Calls) > 3 {
		return nil, 0, nil, fmt.Errorf("app command requires a locator, optional first/last/nth selector, and action")
	}
	action := command.Calls[len(command.Calls)-1]
	target, err := e.elementTarget(command.Calls[:len(command.Calls)-1])
	if err != nil {
		return nil, 0, nil, err
	}
	timeout := e.ActionTimeout
	if command.Expectation {
		timeout = expectationTimeout(action, timeout)
	} else if strings.EqualFold(action.Name, "waitForVisible") || strings.EqualFold(action.Name, "waitForHidden") {
		timeout = timeoutArgument(action, 0, timeout)
	}
	deadlineCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	attempts := 0
	var actual interface{}
	var lastErr error
	for {
		attempts++
		candidate, callErr := e.tryAppAction(deadlineCtx, target, action, command.Expectation)
		if !(deadlineCtx.Err() != nil && callErr != nil && lastErr != nil) {
			actual, lastErr = candidate, callErr
		}
		if lastErr == nil {
			if command.Expectation {
				matched, expected, matchErr := expectationMatches(action, actual)
				if matchErr != nil {
					return actual, attempts, nil, matchErr
				}
				if matched {
					return actual, attempts, passedValidation(source), nil
				}
				lastErr = fmt.Errorf("expectation did not match")
				_ = expected
			} else if waitActionSatisfied(action, actual) {
				return actual, attempts, nil, nil
			} else {
				lastErr = fmt.Errorf("wait condition was not satisfied")
			}
		}
		select {
		case <-deadlineCtx.Done():
			if command.Expectation {
				_, expected, matchErr := expectationMatches(action, actual)
				if matchErr != nil {
					return actual, attempts, nil, matchErr
				}
				return actual, attempts, failedValidation(source, command.Calls[0].Name, expected, actual, attempts, lastErr), nil
			}
			return actual, attempts, nil, fmt.Errorf("action timed out after %s: %w", timeout, lastErr)
		case <-time.After(e.PollInterval):
		}
	}
}

func (e *Executor) executeDevice(ctx context.Context, command *DSLCommand, source string) (interface{}, int, *assertly.Validation, error) {
	if e.ExecuteDevice == nil {
		return nil, 0, nil, fmt.Errorf("device commands are not available")
	}
	if !command.Expectation {
		if len(command.Calls) != 1 {
			return nil, 0, nil, fmt.Errorf("device command requires exactly one action")
		}
		actual, err := e.ExecuteDevice(ctx, e.Session, command.Calls[0])
		return actual, 1, nil, err
	}
	if len(command.Calls) != 2 {
		return nil, 0, nil, fmt.Errorf("device expectation requires one read and one matcher")
	}
	read := command.Calls[0]
	matcher := command.Calls[1]
	timeout := expectationTimeout(matcher, e.ActionTimeout)
	deadlineCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	attempts := 0
	var actual interface{}
	var lastErr error
	for {
		attempts++
		candidate, callErr := e.ExecuteDevice(deadlineCtx, e.Session, read)
		if !(deadlineCtx.Err() != nil && callErr != nil && lastErr != nil) {
			actual, lastErr = candidate, callErr
		}
		if lastErr == nil {
			matched, expected, err := expectationMatches(matcher, actual)
			if err != nil {
				return actual, attempts, nil, err
			}
			if matched {
				return actual, attempts, passedValidation(source), nil
			}
			lastErr = fmt.Errorf("expectation did not match")
			_ = expected
		}
		select {
		case <-deadlineCtx.Done():
			_, expected, err := expectationMatches(matcher, actual)
			if err != nil {
				return actual, attempts, nil, err
			}
			return actual, attempts, failedValidation(source, read.Name, expected, actual, attempts, lastErr), nil
		case <-time.After(e.PollInterval):
		}
	}
}

func (e *Executor) elementTarget(calls []Call) (elementTarget, error) {
	locator, supported, err := e.Resolve(calls[0])
	if err != nil {
		return elementTarget{}, err
	}
	if !supported {
		return elementTarget{}, fmt.Errorf("unsupported locator %q", calls[0].Name)
	}
	target := elementTarget{locator: locator, selection: "strict"}
	if len(calls) == 1 {
		return target, nil
	}
	selection := calls[1]
	switch strings.ToLower(selection.Name) {
	case "first", "last":
		if len(selection.Args) != 0 {
			return elementTarget{}, fmt.Errorf("%s does not accept arguments", selection.Name)
		}
		target.selection = strings.ToLower(selection.Name)
	case "nth":
		index, err := requiredIntArg(selection, 0)
		if err != nil || index < 0 {
			return elementTarget{}, fmt.Errorf("nth requires a non-negative index")
		}
		target.selection = "nth"
		target.index = index
	default:
		return elementTarget{}, fmt.Errorf("unsupported collection operator %q", selection.Name)
	}
	return target, nil
}

func (e *Executor) tryAppAction(ctx context.Context, target elementTarget, action Call, expectation bool) (interface{}, error) {
	elements, err := e.Session.FindElements(ctx, target.locator.Using, target.locator.Value)
	if err != nil {
		return nil, err
	}
	name := strings.ToLower(action.Name)
	switch name {
	case "count", "tohavecount":
		return len(elements), nil
	case "exists", "toexist":
		return len(elements) > 0, nil
	case "waitforhidden", "tobehidden":
		if len(elements) == 0 {
			return true, nil
		}
	}
	elementID, err := selectElement(elements, target)
	if err != nil {
		if (expectation && (name == "tobevisible" || name == "tobeenabled" || name == "tobeselected")) || name == "waitforvisible" {
			return false, nil
		}
		return nil, err
	}
	if name == "waitforhidden" || name == "tobehidden" {
		visible, err := e.Session.ElementBool(ctx, elementID, "displayed")
		return !visible, err
	}
	return e.tryElementAction(ctx, elementID, target, action)
}

func selectElement(elements []string, target elementTarget) (string, error) {
	switch target.selection {
	case "first":
		if len(elements) > 0 {
			return elements[0], nil
		}
	case "last":
		if len(elements) > 0 {
			return elements[len(elements)-1], nil
		}
	case "nth":
		if target.index < len(elements) {
			return elements[target.index], nil
		}
	default:
		if len(elements) == 1 {
			return elements[0], nil
		}
		if len(elements) > 1 {
			return "", fmt.Errorf("locator matched %d elements; use first(), last(), or nth()", len(elements))
		}
	}
	return "", fmt.Errorf("locator matched no element")
}

func (e *Executor) tryElementAction(ctx context.Context, elementID string, target elementTarget, action Call) (interface{}, error) {
	switch strings.ToLower(action.Name) {
	case "tap", "click":
		return nil, e.Session.Click(ctx, elementID)
	case "doubletap":
		return nil, e.doubleTap(ctx, elementID)
	case "longpress":
		duration := optionalIntArg(action, 0, 800)
		return nil, e.longPress(ctx, elementID, duration)
	case "swipe":
		direction, err := requiredStringArg(action, 0)
		if err != nil {
			return nil, err
		}
		percent := optionalFloatArg(action, 1, 0.75)
		return nil, e.swipe(ctx, elementID, direction, percent, 300)
	case "scroll":
		direction, err := requiredStringArg(action, 0)
		if err != nil {
			return nil, err
		}
		count := optionalIntArg(action, 1, 1)
		for i := 0; i < count; i++ {
			if err := e.swipe(ctx, elementID, direction, 0.75, 300); err != nil {
				return nil, err
			}
		}
		return count, nil
	case "dragto":
		return nil, e.dragTo(ctx, elementID, target, action)
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
	case "check", "uncheck":
		selected, err := e.Session.ElementBool(ctx, elementID, "selected")
		if err != nil {
			return nil, err
		}
		shouldSelect := strings.EqualFold(action.Name, "check")
		if selected != shouldSelect {
			err = e.Session.Click(ctx, elementID)
		}
		return shouldSelect, err
	case "text", "tohavetext", "tocontaintext":
		return e.Session.Text(ctx, elementID)
	case "inputvalue", "tohavevalue":
		return e.Session.Attribute(ctx, elementID, "value")
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
	case "rect":
		return e.Session.ElementRect(ctx, elementID)
	default:
		return nil, fmt.Errorf("unsupported element action %q", action.Name)
	}
}

func (e *Executor) doubleTap(ctx context.Context, elementID string) error {
	rect, err := e.Session.ElementRect(ctx, elementID)
	if err != nil {
		return err
	}
	x, y := rectCenter(rect)
	actions := []map[string]interface{}{{
		"type": "pointer", "id": "finger", "parameters": map[string]interface{}{"pointerType": "touch"},
		"actions": []map[string]interface{}{
			{"type": "pointerMove", "duration": 0, "origin": "viewport", "x": x, "y": y},
			{"type": "pointerDown", "button": 0}, {"type": "pointerUp", "button": 0},
			{"type": "pause", "duration": 100},
			{"type": "pointerDown", "button": 0}, {"type": "pointerUp", "button": 0},
		},
	}}
	if err := e.Session.PerformActions(ctx, actions); err != nil {
		return err
	}
	return e.Session.ReleaseActions(ctx)
}

func (e *Executor) longPress(ctx context.Context, elementID string, durationMs int) error {
	rect, err := e.Session.ElementRect(ctx, elementID)
	if err != nil {
		return err
	}
	x, y := rectCenter(rect)
	actions := []map[string]interface{}{{
		"type": "pointer", "id": "finger", "parameters": map[string]interface{}{"pointerType": "touch"},
		"actions": []map[string]interface{}{
			{"type": "pointerMove", "duration": 0, "origin": "viewport", "x": x, "y": y},
			{"type": "pointerDown", "button": 0}, {"type": "pause", "duration": durationMs},
			{"type": "pointerUp", "button": 0},
		},
	}}
	if err := e.Session.PerformActions(ctx, actions); err != nil {
		return err
	}
	return e.Session.ReleaseActions(ctx)
}

func (e *Executor) swipe(ctx context.Context, elementID, direction string, percent float64, durationMs int) error {
	if percent <= 0 || percent > 1 {
		return fmt.Errorf("swipe percent must be in (0,1]")
	}
	rect, err := e.Session.ElementRect(ctx, elementID)
	if err != nil {
		return err
	}
	startX, startY, endX, endY, err := swipePoints(rect, direction, percent)
	if err != nil {
		return err
	}
	return e.Session.PointerGesture(ctx, []PointerPoint{
		{X: startX, Y: startY, Down: true},
		{X: endX, Y: endY, DurationMs: durationMs, Up: true},
	})
}

func (e *Executor) dragTo(ctx context.Context, sourceID string, source elementTarget, action Call) error {
	var target Locator
	var duration int
	switch len(action.Args) {
	case 1, 2:
		value, err := requiredStringArg(action, 0)
		if err != nil {
			return err
		}
		target = Locator{Using: source.locator.Using, Value: value}
		duration = optionalIntArg(action, 1, 500)
	case 3:
		using, err := requiredStringArg(action, 0)
		if err != nil {
			return err
		}
		value, err := requiredStringArg(action, 1)
		if err != nil {
			return err
		}
		target = Locator{Using: using, Value: value}
		duration = optionalIntArg(action, 2, 500)
	default:
		return fmt.Errorf("dragTo expects target value and optional duration, or strategy, value, duration")
	}
	targetIDs, err := e.Session.FindElements(ctx, target.Using, target.Value)
	if err != nil {
		return err
	}
	targetID, err := selectElement(targetIDs, elementTarget{selection: "strict"})
	if err != nil {
		return fmt.Errorf("drag target: %w", err)
	}
	from, err := e.Session.ElementRect(ctx, sourceID)
	if err != nil {
		return err
	}
	to, err := e.Session.ElementRect(ctx, targetID)
	if err != nil {
		return err
	}
	fromX, fromY := rectCenter(from)
	toX, toY := rectCenter(to)
	return e.Session.PointerGesture(ctx, []PointerPoint{
		{X: fromX, Y: fromY, Down: true},
		{X: toX, Y: toY, DurationMs: duration, Up: true},
	})
}

func rectCenter(rect Rect) (int, int) {
	return int(math.Round(rect.X + rect.Width/2)), int(math.Round(rect.Y + rect.Height/2))
}

func swipePoints(rect Rect, direction string, percent float64) (int, int, int, int, error) {
	cx, cy := rectCenter(rect)
	dx := int(math.Round(rect.Width * percent / 2))
	dy := int(math.Round(rect.Height * percent / 2))
	switch strings.ToLower(direction) {
	case "up":
		return cx, cy + dy, cx, cy - dy, nil
	case "down":
		return cx, cy - dy, cx, cy + dy, nil
	case "left":
		return cx + dx, cy, cx - dx, cy, nil
	case "right":
		return cx - dx, cy, cx + dx, cy, nil
	default:
		return 0, 0, 0, 0, fmt.Errorf("direction must be up, down, left, or right")
	}
}

func waitActionSatisfied(action Call, actual interface{}) bool {
	switch strings.ToLower(action.Name) {
	case "waitforvisible", "waitforhidden":
		value, _ := actual.(bool)
		return value
	default:
		return true
	}
}

func expectationMatches(call Call, actual interface{}) (bool, interface{}, error) {
	switch strings.ToLower(call.Name) {
	case "tobevisible", "tobeenabled", "tobeselected", "toexist", "tobehidden", "tohavealert":
		value, ok := actual.(bool)
		if strings.EqualFold(call.Name, "toHaveAlert") && !ok {
			value = strings.TrimSpace(fmt.Sprint(actual)) != ""
			ok = true
		}
		return ok && value, true, nil
	case "tohavetext", "tohavevalue", "tohaveorientation", "tohavecontext", "tobe", "toequal":
		expected, err := requiredArg(call, 0)
		return fmt.Sprint(actual) == fmt.Sprint(expected), expected, err
	case "tocontaintext", "tocontain":
		expected, err := requiredArg(call, 0)
		return strings.Contains(fmt.Sprint(actual), fmt.Sprint(expected)), expected, err
	case "tohaveattribute":
		if len(call.Args) < 2 {
			return false, nil, fmt.Errorf("toHaveAttribute requires name and value")
		}
		return fmt.Sprint(actual) == fmt.Sprint(call.Args[1]), call.Args[1], nil
	case "tohavecount":
		expected, err := requiredIntArg(call, 0)
		actualCount, ok := asInt(actual)
		return err == nil && ok && actualCount == expected, expected, err
	default:
		return false, nil, fmt.Errorf("unsupported expectation %q", call.Name)
	}
}

func expectationTimeout(call Call, fallback time.Duration) time.Duration {
	index := -1
	switch strings.ToLower(call.Name) {
	case "tobevisible", "tobeenabled", "tobeselected", "toexist", "tobehidden", "tohavealert":
		index = 0
	case "tohavetext", "tocontaintext", "tohavevalue", "tohavecount", "tohaveorientation", "tohavecontext", "tobe", "toequal", "tocontain":
		index = 1
	case "tohaveattribute":
		index = 2
	}
	if index < 0 || index >= len(call.Args) {
		return fallback
	}
	return timeoutArgument(call, index, fallback)
}

func timeoutArgument(call Call, index int, fallback time.Duration) time.Duration {
	if index < 0 || index >= len(call.Args) {
		return fallback
	}
	milliseconds, ok := asInt(call.Args[index])
	if !ok || milliseconds <= 0 {
		return fallback
	}
	return time.Duration(milliseconds) * time.Millisecond
}

func passedValidation(source string) *assertly.Validation {
	validation := assertly.NewValidation()
	validation.Description = source
	validation.PassedCount = 1
	return validation
}

func failedValidation(source, path string, expected, actual interface{}, attempts int, lastErr error) *assertly.Validation {
	validation := assertly.NewValidation()
	validation.Description = source
	failure := assertly.NewFailure("mobile", path, assertly.EqualViolation, expected, actual)
	failure.Message = fmt.Sprintf("%s after %d attempt(s); last error: %v", failure.Message, attempts, lastErr)
	validation.AddFailure(failure)
	return validation
}

func requiredArg(call Call, index int) (interface{}, error) {
	if index >= len(call.Args) {
		return nil, fmt.Errorf("%s requires argument %d", call.Name, index+1)
	}
	return call.Args[index], nil
}

func requiredStringArg(call Call, index int) (string, error) {
	value, err := requiredArg(call, index)
	if err != nil {
		return "", err
	}
	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s argument %d must be a string", call.Name, index+1)
	}
	return text, nil
}

func requiredIntArg(call Call, index int) (int, error) {
	value, err := requiredArg(call, index)
	if err != nil {
		return 0, err
	}
	integer, ok := asInt(value)
	if !ok {
		return 0, fmt.Errorf("%s argument %d must be an integer", call.Name, index+1)
	}
	return integer, nil
}

func optionalIntArg(call Call, index, fallback int) int {
	if index >= len(call.Args) {
		return fallback
	}
	value, ok := asInt(call.Args[index])
	if !ok {
		return fallback
	}
	return value
}

func optionalFloatArg(call Call, index int, fallback float64) float64 {
	if index >= len(call.Args) {
		return fallback
	}
	switch value := call.Args[index].(type) {
	case float64:
		return value
	case int64:
		return float64(value)
	case int:
		return float64(value)
	case string:
		parsed, err := strconv.ParseFloat(value, 64)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func asInt(value interface{}) (int, bool) {
	switch actual := value.(type) {
	case int:
		return actual, true
	case int64:
		return int(actual), true
	case float64:
		return int(actual), actual == math.Trunc(actual)
	case string:
		parsed, err := strconv.Atoi(actual)
		return parsed, err == nil
	default:
		return 0, false
	}
}
