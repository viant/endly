package mobile

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type Call struct {
	Name string
	Args []interface{}
}

type DSLCommand struct {
	Key         string
	Namespace   string
	Calls       []Call
	Expectation bool
}

type TypedLocator struct {
	Strategy  string `json:"strategy"`
	Value     string `json:"value"`
	Selection string `json:"selection,omitempty"`
	Index     int    `json:"index,omitempty"`
}

type TypedCall struct {
	Name string        `json:"name"`
	Args []interface{} `json:"args,omitempty"`
}

type TypedExpectation struct {
	Locator   *TypedLocator `json:"locator,omitempty"`
	Device    *TypedCall    `json:"device,omitempty"`
	Matcher   TypedCall     `json:"matcher"`
	TimeoutMs int           `json:"timeoutMs,omitempty"`
}

// TypedCommand is the structured alternative to the textual DSL. Exactly one
// of Locator+Action, Device, or Expect must be configured.
type TypedCommand struct {
	Key     string            `json:"key,omitempty"`
	Locator *TypedLocator     `json:"locator,omitempty"`
	Action  *TypedCall        `json:"action,omitempty"`
	Device  *TypedCall        `json:"device,omitempty"`
	Expect  *TypedExpectation `json:"expect,omitempty"`
}

func ParseInstruction(candidate interface{}) (*DSLCommand, string, error) {
	switch actual := candidate.(type) {
	case string:
		parsed, err := ParseDSL(actual)
		return parsed, actual, err
	case *DSLCommand:
		if actual == nil {
			return nil, "", fmt.Errorf("command was nil")
		}
		return actual, fmt.Sprintf("%v", actual), nil
	case DSLCommand:
		return &actual, fmt.Sprintf("%v", actual), nil
	case *TypedCommand:
		if actual == nil {
			return nil, "", fmt.Errorf("typed command was nil")
		}
		return actual.DSL()
	case TypedCommand:
		return actual.DSL()
	default:
		data, err := json.Marshal(candidate)
		if err != nil {
			return nil, "", fmt.Errorf("encode typed command: %w", err)
		}
		command := &TypedCommand{}
		if err := json.Unmarshal(data, command); err != nil {
			return nil, "", fmt.Errorf("decode typed command: %w", err)
		}
		parsed, _, err := command.DSL()
		return parsed, string(data), err
	}
}

func (c *TypedCommand) DSL() (*DSLCommand, string, error) {
	data, _ := json.Marshal(c)
	source := string(data)
	configured := 0
	if c.Locator != nil || c.Action != nil {
		configured++
	}
	if c.Device != nil {
		configured++
	}
	if c.Expect != nil {
		configured++
	}
	if configured != 1 {
		return nil, source, fmt.Errorf("typed command requires exactly one locator action, device call, or expectation")
	}
	if c.Locator != nil || c.Action != nil {
		if c.Locator == nil || c.Action == nil {
			return nil, source, fmt.Errorf("typed locator command requires Locator and Action")
		}
		calls, err := typedLocatorCalls(c.Locator)
		if err != nil {
			return nil, source, err
		}
		if err := validateTypedCall(*c.Action); err != nil {
			return nil, source, err
		}
		calls = append(calls, Call{Name: c.Action.Name, Args: c.Action.Args})
		return &DSLCommand{Key: c.Key, Namespace: "app", Calls: calls}, source, nil
	}
	if c.Device != nil {
		if err := validateTypedCall(*c.Device); err != nil {
			return nil, source, err
		}
		return &DSLCommand{Key: c.Key, Namespace: "device", Calls: []Call{{Name: c.Device.Name, Args: c.Device.Args}}}, source, nil
	}
	expect := c.Expect
	if expect.Matcher.Name == "" {
		return nil, source, fmt.Errorf("typed expectation matcher name is required")
	}
	matcherArgs := append([]interface{}{}, expect.Matcher.Args...)
	if expect.TimeoutMs > 0 {
		matcherArgs = append(matcherArgs, int64(expect.TimeoutMs))
	}
	matcher := Call{Name: expect.Matcher.Name, Args: matcherArgs}
	if expect.Locator != nil && expect.Device != nil {
		return nil, source, fmt.Errorf("typed expectation Locator and Device are mutually exclusive")
	}
	if expect.Locator != nil {
		calls, err := typedLocatorCalls(expect.Locator)
		if err != nil {
			return nil, source, err
		}
		calls = append(calls, matcher)
		return &DSLCommand{Key: c.Key, Namespace: "app", Calls: calls, Expectation: true}, source, nil
	}
	if expect.Device != nil {
		if err := validateTypedCall(*expect.Device); err != nil {
			return nil, source, err
		}
		return &DSLCommand{Key: c.Key, Namespace: "device", Calls: []Call{{Name: expect.Device.Name, Args: expect.Device.Args}, matcher}, Expectation: true}, source, nil
	}
	return nil, source, fmt.Errorf("typed expectation requires Locator or Device")
}

func typedLocatorCalls(locator *TypedLocator) ([]Call, error) {
	if locator == nil || strings.TrimSpace(locator.Strategy) == "" || strings.TrimSpace(locator.Value) == "" {
		return nil, fmt.Errorf("typed locator strategy and value are required")
	}
	result := []Call{{Name: "locator", Args: []interface{}{locator.Strategy, locator.Value}}}
	switch strings.ToLower(locator.Selection) {
	case "":
	case "first", "last":
		result = append(result, Call{Name: locator.Selection})
	case "nth":
		if locator.Index < 0 {
			return nil, fmt.Errorf("typed locator nth index must be non-negative")
		}
		result = append(result, Call{Name: "nth", Args: []interface{}{int64(locator.Index)}})
	default:
		return nil, fmt.Errorf("unsupported typed locator selection %q", locator.Selection)
	}
	return result, nil
}

func validateTypedCall(call TypedCall) error {
	if !isIdentifier(call.Name) {
		return fmt.Errorf("invalid typed call name %q", call.Name)
	}
	return nil
}

func ParseDSL(source string) (*DSLCommand, error) {
	expression := strings.TrimSpace(source)
	if expression == "" {
		return nil, fmt.Errorf("command was empty")
	}
	result := &DSLCommand{}
	if index := topLevelAssignment(expression); index >= 0 {
		result.Key = strings.TrimSpace(expression[:index])
		if !isIdentifier(result.Key) {
			return nil, fmt.Errorf("invalid result key %q", result.Key)
		}
		expression = strings.TrimSpace(expression[index+1:])
	}
	if strings.HasPrefix(strings.ToLower(expression), "expect(") {
		inner, rest, err := consumeBalanced(expression[len("expect"):])
		if err != nil {
			return nil, err
		}
		base, err := parseInvocation(inner)
		if err != nil {
			return nil, fmt.Errorf("parse expectation target: %w", err)
		}
		matcher, err := parseChain(strings.TrimPrefix(strings.TrimSpace(rest), "."))
		if err != nil || len(matcher) != 1 {
			return nil, fmt.Errorf("expectation requires exactly one matcher")
		}
		base.Calls = append(base.Calls, matcher[0])
		base.Expectation = true
		return base, nil
	}
	parsed, err := parseInvocation(expression)
	if err != nil {
		return nil, err
	}
	parsed.Key = result.Key
	return parsed, nil
}

func parseInvocation(expression string) (*DSLCommand, error) {
	dot := strings.IndexByte(expression, '.')
	if dot <= 0 {
		return nil, fmt.Errorf("expected app.* or device.* invocation")
	}
	namespace := strings.ToLower(strings.TrimSpace(expression[:dot]))
	if namespace != "app" && namespace != "device" {
		return nil, fmt.Errorf("unsupported namespace %q", namespace)
	}
	calls, err := parseChain(expression[dot+1:])
	if err != nil {
		return nil, err
	}
	if len(calls) == 0 {
		return nil, fmt.Errorf("invocation contained no calls")
	}
	return &DSLCommand{Namespace: namespace, Calls: calls}, nil
}

func parseChain(source string) ([]Call, error) {
	remaining := strings.TrimSpace(source)
	result := []Call{}
	for remaining != "" {
		open := strings.IndexByte(remaining, '(')
		if open <= 0 {
			return nil, fmt.Errorf("expected call in %q", remaining)
		}
		name := strings.TrimSpace(remaining[:open])
		if !isIdentifier(name) {
			return nil, fmt.Errorf("invalid call name %q", name)
		}
		inner, rest, err := consumeBalanced(remaining[open:])
		if err != nil {
			return nil, err
		}
		args, err := parseArguments(inner)
		if err != nil {
			return nil, fmt.Errorf("parse %s arguments: %w", name, err)
		}
		result = append(result, Call{Name: name, Args: args})
		remaining = strings.TrimSpace(rest)
		if remaining == "" {
			break
		}
		if remaining[0] != '.' {
			return nil, fmt.Errorf("unexpected suffix %q", remaining)
		}
		remaining = strings.TrimSpace(remaining[1:])
	}
	return result, nil
}

func consumeBalanced(source string) (string, string, error) {
	source = strings.TrimSpace(source)
	if source == "" || source[0] != '(' {
		return "", "", fmt.Errorf("expected opening parenthesis")
	}
	quote := byte(0)
	escaped := false
	depth := 0
	for i := 0; i < len(source); i++ {
		char := source[i]
		if escaped {
			escaped = false
			continue
		}
		if quote != 0 {
			if char == '\\' {
				escaped = true
			} else if char == quote {
				quote = 0
			}
			continue
		}
		if char == '\'' || char == '"' {
			quote = char
			continue
		}
		switch char {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return strings.TrimSpace(source[1:i]), strings.TrimSpace(source[i+1:]), nil
			}
		}
	}
	return "", "", fmt.Errorf("unterminated parenthesis")
}

func parseArguments(source string) ([]interface{}, error) {
	if strings.TrimSpace(source) == "" {
		return nil, nil
	}
	parts := []string{}
	start := 0
	quote := byte(0)
	escaped := false
	depth := 0
	for i := 0; i < len(source); i++ {
		char := source[i]
		if escaped {
			escaped = false
			continue
		}
		if quote != 0 {
			if char == '\\' {
				escaped = true
			} else if char == quote {
				quote = 0
			}
			continue
		}
		if char == '\'' || char == '"' {
			quote = char
			continue
		}
		switch char {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(source[start:i]))
				start = i + 1
			}
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quoted argument")
	}
	parts = append(parts, strings.TrimSpace(source[start:]))
	result := make([]interface{}, 0, len(parts))
	for _, part := range parts {
		value, err := parseArgument(part)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func parseArgument(value string) (interface{}, error) {
	if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') {
		if value[len(value)-1] != value[0] {
			return nil, fmt.Errorf("unterminated string %q", value)
		}
		if value[0] == '"' {
			return strconv.Unquote(value)
		}
		inner := strings.ReplaceAll(value[1:len(value)-1], `\'`, `'`)
		return strings.ReplaceAll(inner, `\\`, `\`), nil
	}
	switch strings.ToLower(value) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "null":
		return nil, nil
	}
	if integer, err := strconv.ParseInt(value, 10, 64); err == nil {
		return integer, nil
	}
	if number, err := strconv.ParseFloat(value, 64); err == nil {
		return number, nil
	}
	return value, nil
}

func topLevelAssignment(source string) int {
	quote := byte(0)
	escaped := false
	depth := 0
	for i := 0; i < len(source); i++ {
		char := source[i]
		if escaped {
			escaped = false
			continue
		}
		if quote != 0 {
			if char == '\\' {
				escaped = true
			} else if char == quote {
				quote = 0
			}
			continue
		}
		if char == '\'' || char == '"' {
			quote = char
			continue
		}
		switch char {
		case '(':
			depth++
		case ')':
			depth--
		case '=':
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func isIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for i, char := range value {
		if !(unicode.IsLetter(char) || char == '_' || (i > 0 && unicode.IsDigit(char))) {
			return false
		}
	}
	return true
}
