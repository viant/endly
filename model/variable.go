package model

import (
	"fmt"
	"github.com/viant/endly/model/criteria"
	"github.com/viant/endly/model/criteria/eval"
	"github.com/viant/endly/model/criteria/parser"
	"github.com/viant/endly/model/yml"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"gopkg.in/yaml.v3"
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
	if from != "" {
		if strings.Contains(from, "$") {
			value, _ = in.GetValue(in.ExpandAsText(from))
		} else {
			value, _ = in.GetValue(from)
		}
	}
	return value, nil
}

func (v *Variable) formatStateInfo(state data.Map) string {
	var aMap = state.AsEncodableMap()
	var result = map[string]interface{}{}
	for k, v := range aMap {
		if "func()" == v {
			continue
		}
		if toolbox.IsStruct(result) || toolbox.IsSlice(result) {
			JSONVal, _ := toolbox.AsJSONText(v)
			if len(JSONVal) < 250 {
				result[k] = v
			} else if toolbox.IsSlice(result) {
				result[k] = "[..large array..]"
			} else {
				result[k] = "{..large object..}"
			}
			continue
		}
		result[k] = v
	}
	JSONResult, _ := toolbox.AsIndentJSONText(result)
	return JSONResult
}

func (v *Variable) validate(value interface{}, in data.Map) error {
	val := toolbox.AsString(value)
	if v.Required {
		source := in.GetString(OwnerURL)
		if value == nil || toolbox.AsString(value) == "" {
			return fmt.Errorf("variable '%v' is required by %v, but was empty,  \n\tstate dump: %v", v.Name, source, v.formatStateInfo(in))
		}
		if v.EmptyIfUnexpanded && val == v.Value && strings.HasPrefix(val, "$") {
			return fmt.Errorf("variable '%v' is required by %v, but %v was not expanded,\n\tstate dump:%v", v.Name, source, v.Value, v.formatStateInfo(in))
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
	}
	return value
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
