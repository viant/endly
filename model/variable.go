package model

import (
	"bytes"
	"fmt"
	"github.com/viant/endly/model/criteria"
	"github.com/viant/neatly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

//Variable represents a variable
type Variable struct {
	Name              string            `description:"name"`
	Value             interface{}       `description:"default value"`
	From              string            `description:"context state map key to pull data"`
	When              string            `description:"criteria if specified this variable will be set only if evaluated criteria is true (it can use $in, and $out state variables)"`
	Else              interface{}       `description:"if when criteria is not met then else can provide variable value alternative"`
	Persist           bool              `description:"stores in tmp directory to be used as backup if data is not in the cotnext"`
	Required          bool              `description:"flag that validates that from returns non empty value or error is generated"`
	EmptyIfUnexpanded bool              `description:"threat variable value empty if it was not expanded"`
	Replace           map[string]string `description:"replacements map, if key if specified substitute variable value with corresponding value. This will work only for string replacements"`
}

func (v *Variable) tempfile() string {
	return path.Join(os.Getenv("TMPDIR"), v.Name+".var")
}

//PersistValue persist variable
func (v *Variable) PersistValue() error {
	if v.Value != nil {
		var filename = v.tempfile()
		_ = toolbox.RemoveFileIfExist(filename)
		file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer file.Close()
		if err = toolbox.NewJSONEncoderFactory().Create(file).Encode(v.Value); err != nil {
			return err
		}
	}
	return nil
}

//Load loads persisted variable value.
func (v *Variable) Load() error {
	var err error
	var encoded []byte
	if v.Value == nil {
		var filename = v.tempfile()
		if !toolbox.FileExists(filename) {
			return nil
		}
		encoded, _ = ioutil.ReadFile(filename)
		err = toolbox.NewJSONDecoderFactory().Create(bytes.NewReader(encoded)).Decode(&v.Value)
	}
	return err
}

func (v *Variable) fromVariable() *Variable {
	var fromExpr = v.From
	fromExpr = strings.Replace(fromExpr, "<-", "", 1)
	dotPosition := strings.Index(fromExpr, ".")
	if dotPosition != -1 {
		fromExpr = string(fromExpr[:dotPosition])
	}
	return &Variable{
		Name: fromExpr,
	}
}

func (v *Variable) getValueFromInput(in data.Map) (interface{}, error) {
	var value interface{}
	from := v.From
	if from == "" && v.Value == nil {
		from = v.Name
	}
	if from != "" {
		var has bool

		if strings.Contains(from, "$") {
			value, has = in.GetValue(in.ExpandAsText(from))
		} else {
			value, has = in.GetValue(from)
		}
		if !has {
			fromVariable := v.fromVariable()
			err := fromVariable.Load()
			if fromVariable.Value != nil {
				in.SetValue(fromVariable.Name, fromVariable.Value)
				value, _ = in.GetValue(from)
			}
			if err != nil {
				return err, nil
			}
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
		source := in.GetString(neatly.OwnerURL)
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
	result, _ := criteria.Evaluate(nil, state, v.When, "", false)
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

func (v *Variable) replaceValue(value interface{}) interface{} {
	if len(v.Replace) == 0 {
		return value
	}
	text, ok := value.(string)
	if !ok {
		return value
	}

	for k, v := range v.Replace {
		text = strings.Replace(text, k, v, 1)
	}
	return text
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

	value = v.replaceValue(value)
	if v.Name != "" {
		out.SetValue(v.Name, value)
	}
	return nil
}

//NewVariable creates a new variable
func NewVariable(name, form, when string, required bool, value, elseValue interface{}, replace map[string]string, emptyIfUnexpanded bool) *Variable {
	return &Variable{
		Name:              name,
		From:              form,
		When:              when,
		Required:          required,
		Value:             value,
		Else:              elseValue,
		Replace:           replace,
		EmptyIfUnexpanded: emptyIfUnexpanded,
	}
}

//Variables a slice of variables
type Variables []*Variable

//Apply evaluates all variable from in map to out map
func (v *Variables) Apply(in, out data.Map) error {
	if out == nil {
		return fmt.Errorf("out state was empty")
	}
	if in == nil {
		in = data.NewMap()
	}
	for _, variable := range *v {
		if variable == nil {
			continue
		}
		if err := variable.Apply(in, out); err != nil {
			return err
		}
	}
	return nil
}

//String returns a variable info
func (v Variables) String() string {
	var result = ""
	for _, item := range v {
		if item == nil {
			continue
		}
		result += fmt.Sprintf("{Name:%v From:%v Value:%v},", item.Name, item.From, item.Value)
	}
	return result
}

//VariableExpression represent a variable expression [!] VariableName = [when  ?] value : otherwiseValue,
// exclamation mark flags variable as required
type VariableExpression string

//AsVariable converts expression to variable
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
	if toolbox.IsCompleteJSON(value) {
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
	if !hasConditionalAssignment(value) {
		variable.Value = normalizeVariableValue(value)
		return
	}
	var whenIndex = strings.Index(value, "?")
	if whenIndex != -1 {
		predicate := strings.TrimSpace(string(value[:whenIndex]))
		value := strings.TrimSpace(string(value[whenIndex+1:]))
		variable.When = predicate
		variable.Value = value
		elseIndex := strings.LastIndex(value, ":")
		if elseIndex != -1 {
			variable.Value = string(value[:elseIndex])
			variable.Else = normalizeVariableValue(string(value[elseIndex+1:]))
		}
	}
	variable.Value = normalizeVariableValue(toolbox.AsString(variable.Value))
}

func isValidPredicate(candidate string) bool {
	if !strings.Contains(candidate, "$") {
		return false
	}
	_, err := criteria.NewParser().Parse(candidate)
	return err == nil
}

func hasConditionalAssignment(candidate string) bool {
	questionMarkCount := strings.Count(candidate, "?")
	if questionMarkCount != 1 {
		return false
	}
	parts := strings.SplitN(candidate, "?", 2)
	if !isValidPredicate(parts[0]) {
		return false
	}
	elseCount := strings.Count(candidate, ":")
	if elseCount == 0 {
		return true
	}
	return strings.Index(candidate, "?") < strings.LastIndex(candidate, ":")
}
