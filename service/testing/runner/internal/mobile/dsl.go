package mobile

import (
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
