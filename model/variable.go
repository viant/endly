package model

import (
	"fmt"
	"github.com/viant/endly/internal/util"
	"github.com/viant/endly/model/criteria"
	"github.com/viant/endly/model/criteria/eval"
	"github.com/viant/endly/model/criteria/parser"
	"github.com/viant/endly/model/graph/yml"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"gopkg.in/yaml.v3"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

const (
	OwnerURL = "ownerURL"
)

func (v *Variable) MarshalYAML() (interface{}, error) {
	type variable Variable

	customVar := variable(*v)
	orig := &yaml.Node{}
	err := orig.Encode(&customVar)
	if err != nil {
		return nil, err
	}
	value := yml.Nodes(orig.Content).LookupValueNode("value")
	result := yaml.Node{
		Kind:    yaml.MappingNode,
		Tag:     orig.Tag,
		Content: []*yaml.Node{},
	}
	result.Content = yml.Nodes(result.Content).AppendScalar(v.Name)
	if v.When != "" {
		textValue := v.When + " ? " + toolbox.AsString(v.Value)
		if v.Else != nil {
			textValue += " : " + toolbox.AsString(v.Else)
		}
		value.Value = textValue
	}
	result.Content = append(result.Content, value)
	return &result, err
}

// Variable represents a variable
type Variable struct {
	Name              string            `description:"name"`
	Value             interface{}       `description:"default value"`
	From              string            `description:"context state map key to pull data"`
	When              string            `description:"criteria if specified this variable will be set only if evaluated criteria is true (it can use $in, and $out state variables)"`
	Else              interface{}       `description:"if when criteria is not met then else can provide variable value alternative"`
	Required          bool              `description:"flag that validates that from returns non empty value or error is generated"`
	EmptyIfUnexpanded bool              `description:"threat variable value empty if it was not expanded"`
	Replace           map[string]string `description:"replacements map, if key if specified substitute variable value with corresponding value. This will work only for string replacements"`
	whenEval          eval.Compute
}

func (v *Variable) getValueFromInput(in data.Map) (interface{}, error) {
	var value interface{}
	from := v.From
	if from == "" && v.Value == nil {
		from = v.Name
	}
	if from == "" && v.When != "" {
		from = v.Name
	}
	if from != "" {
		if strings.Contains(from, "$") {
			value, _ = in.GetValue(in.ExpandAsText(from))
		} else {
			value, _ = in.GetValue(from)
		}
	}
	return value, nil
}

// Required-variable diagnostics must never encode or traverse state values.
// Keep the lexically first keys with bounded memory, even for a huge state map.
func (v *Variable) formatAvailableKeys(state data.Map) string {
	const maxKeys = 16
	keys := make([]string, 0, maxKeys+1)
	for key := range state {
		if !diagnosticKey(key) {
			continue
		}
		if len(keys) == maxKeys && key >= keys[len(keys)-1] {
			continue
		}
		keys = append(keys, key)
		sort.Strings(keys)
		if len(keys) > maxKeys {
			keys = keys[:maxKeys]
		}
	}
	result := strings.Join(keys, ", ")
	if omitted := len(state) - len(keys); omitted > 0 {
		if result != "" {
			result += "; "
		}
		result += strconv.Itoa(omitted) + " more keys omitted"
	}
	return "[" + result + "]"
}

func diagnosticKey(key string) bool {
	if len(key) == 0 || len(key) > 48 {
		return false
	}
	for _, c := range key {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '.' || c == '-') {
			return false
		}
	}
	return true
}

func boundedDiagnosticName(name string) string {
	if len(name) > 64 {
		name = name[:64] + "..."
	}
	return strings.Map(func(c rune) rune {
		if c < 32 || c > 126 || c == '\'' || c == '"' {
			return '_'
		}
		return c
	}, name)
}

func diagnosticOwner(state data.Map) string {
	raw, _ := state[OwnerURL].(string)
	if raw == "" || len(raw) > 2048 {
		return "(workflow unavailable)"
	}
	owner, err := url.Parse(raw)
	if err != nil || owner.Opaque != "" {
		return "(workflow unavailable)"
	}
	owner.User = nil
	owner.RawQuery = ""
	owner.ForceQuery = false
	owner.Fragment = ""
	owner.RawFragment = ""
	result := owner.String()
	result = strings.Map(func(c rune) rune {
		if c < 32 || c > 126 {
			return '_'
		}
		return c
	}, result)
	if len(result) > 192 {
		result = result[:192] + "..."
	}
	return result
}

func (v *Variable) validate(value interface{}, in data.Map) error {
	val := toolbox.AsString(value)
	if v.Required {
		source := diagnosticOwner(in)
		if value == nil || toolbox.AsString(value) == "" {
			return fmt.Errorf("variable '%s' is required by %s, but was empty; supply the required workflow parameter; available keys: %s", boundedDiagnosticName(v.Name), source, v.formatAvailableKeys(in))
		}
		if v.EmptyIfUnexpanded && val == v.Value && strings.HasPrefix(val, "$") {
			return fmt.Errorf("variable '%s' is required by %s, but its default expression was not expanded; supply the required workflow parameter or its referenced input; available keys: %s", boundedDiagnosticName(v.Name), source, v.formatAvailableKeys(in))
		}
	}
	return nil
}

func (v *Variable) canApply(in, out data.Map) bool {
	var state data.Map = map[string]interface{}{
		"in":  in,
		"out": out,
	}
	for k, v := range in {
		state[k] = v
	}
	result, _ := criteria.Evaluate(nil, state, v.When, &v.whenEval, "", false)
	return result
}

func (v *Variable) getValue(in data.Map) interface{} {
	value := v.Value
	if value != nil {
		value = in.Expand(value)
		// UDF-produced data follows the same native expansion path as a
		// referenced map/list, once, using the current invocation state.
		if source, ok := v.Value.(string); ok && isSingleExpression(source, true) && util.IsContainer(value) {
			value = util.ExpandReturnedContainer(in, value)
		}
		// Resolve one scalar expression returned by explicit variable indirection
		// at invocation time. Workflow compilation must not freeze task outputs.
		if source, ok := v.Value.(string); ok && isSingleVariableReference(source) {
			if scalar, ok := value.(string); ok && scalar != source && strings.HasPrefix(scalar, "$") {
				value = in.Expand(scalar)
			}
		}
	}
	return value
}

// isSingleVariableReference uses native expression boundaries for both $name
// and ${name}; interpolated strings and UDF calls are not variable references.
func isSingleVariableReference(source string) bool {
	return isSingleExpression(source, false)
}

func isSingleExpression(source string, udf bool) bool {
	return util.IsSingleExpression(source, udf)
}

func (v *Variable) getElse(in data.Map) interface{} {
	value := v.Else
	if value != nil {
		value = in.Expand(value)
	}
	return value
}

func (v *Variable) applyElse(in, out data.Map) interface{} {
	value := v.getElse(in)
	if v.Name != "" {
		out.SetValue(v.Name, value)
	}
	return value
}

func (v *Variable) Apply(in, out data.Map) error {
	if v.When != "" {
		if !v.canApply(in, out) {
			value := v.applyElse(in, out)
			return v.validate(value, in)
		}
	}
	value, err := v.getValueFromInput(in)
	if err != nil {
		return err
	}

	if value == nil || (v.Required && toolbox.AsString(value) == "") {
		value = v.getValue(in)
	}

	if err := v.validate(value, in); err != nil {
		return err
	}
	if v.Name != "" {
		out.SetValue(v.Name, value)
	}
	return nil
}

// NewVariable creates a new variable
func NewVariable(name, form, when string, required bool, value, elseValue interface{}, replace map[string]string, emptyIfUnexpanded bool) *Variable {
	return &Variable{
		Name:     name,
		From:     form,
		When:     when,
		Required: required,
		Value:    value,
		Else:     elseValue,
	}
}

// VariableExpression represent a variable expression [!] VariableName = [when  ?] value : otherwiseValue,
// exclamation mark flags variable as required
type VariableExpression string

// AsVariable converts expression to variable
func (e *VariableExpression) AsVariable() (*Variable, error) {
	var value = strings.TrimSpace(string(*e))
	var result = &Variable{}
	pair := strings.SplitN(value, "=", 2)
	if len(pair) < 2 {
		return nil, fmt.Errorf("expected variable declaration but had: %v", value)
	}
	extractFromKey(pair[0], result)
	extractFromValue(pair[1], result)
	return result, nil
}

func normalizeVariableValue(value string) interface{} {
	value = strings.TrimSpace(value)
	if "nil" == value {
		return nil
	}
	if strings.HasPrefix(value, "'") {
		return strings.Trim(value, "'")
	}
	if toolbox.IsStructuredJSON(value) {
		if JSON, err := toolbox.JSONToInterface(value); err == nil {
			return JSON
		}
	}
	return value
}

func extractFromKey(key string, variable *Variable) {
	isRequired := strings.HasPrefix(key, "!")
	if isRequired {
		key = string(key[1:])
	}
	variable.Name = strings.TrimSpace(key)
	variable.Required = isRequired
	variable.EmptyIfUnexpanded = isRequired
}

func extractFromValue(value string, variable *Variable) {
	when, whenExpr, elseExpr, _ := parser.ParseDeclaration(value)
	if when == "" {
		variable.Value = normalizeVariableValue(value)
		return
	}
	variable.When = when
	variable.Value = whenExpr
	variable.Else = elseExpr
	variable.Value = normalizeVariableValue(toolbox.AsString(variable.Value))
}
