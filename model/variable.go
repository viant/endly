package model

import (
	"bytes"
	"fmt"
	"github.com/viant/endly/criteria"
	"github.com/viant/endly/util"
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
		toolbox.RemoveFileIfExist(filename)
		file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer file.Close()
		toolbox.NewJSONEncoderFactory().Create(file).Encode(v.Value)
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

func normalizeValue(value string) interface{} {
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

//AsVariable converts expression to variable
func (e *VariableExpression) AsVariable() (*Variable, error) {
	var value = strings.TrimSpace(string(*e))
	isRequired := strings.HasPrefix(value, "!")
	if isRequired {
		value = string(value[1:])
	}
	pair := strings.Split(value, "=")
	if len(pair) != 2 {
		return nil, fmt.Errorf("invalid variable expression, expected '=' operator")
	}
	var result = &Variable{
		Required:          isRequired,
		EmptyIfUnexpanded: isRequired,
	}

	result.Name = strings.TrimSpace(pair[0])
	var whenIndex = strings.Index(pair[1], "?")
	if whenIndex != -1 {
		result.When = strings.TrimSpace(string(pair[1][:whenIndex]))
		value := strings.TrimSpace(string(pair[1][whenIndex+1:]))
		result.Value = value
		elseIndex := strings.LastIndex(value, ":")
		if elseIndex != -1 {
			result.Value = string(value[:elseIndex])
			result.Else = normalizeValue(string(value[elseIndex+1:]))
		}
	} else {
		result.Name = strings.TrimSpace(pair[0])
		result.Value = strings.TrimSpace(pair[1])
	}
	result.Value = normalizeValue(toolbox.AsString(result.Value))
	return result, nil
}

//GetVariables returns variables from Variables ([]*Variable), []string (as expression) or from []interface{} (where interface is a map matching Variable struct)
func GetVariables(baseURLs []string, source interface{}) (Variables, error) {
	if source == nil {
		return nil, nil
	}
	switch value := source.(type) {
	case *Variables:
		return *value, nil
	case Variables:
		return value, nil
	case string:
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, nil
		}
		var result Variables = make([]*Variable, 0)
		loaded, err := util.LoadData(baseURLs, value)
		if err == nil {
			toolbox.DefaultConverter.AssignConverted(&result, loaded)
		}

		return result, err
	}

	var result Variables = make([]*Variable, 0)
	if !toolbox.IsSlice(source) {
		return nil, fmt.Errorf("invalid varaibles type: %T, expected %T or %T", source, result, []string{})
	}

	if _, err := util.NormalizeMap(source, false); err == nil {
		toolbox.ProcessMap(source, func(key, value interface{}) bool {
			variable, e := newVariableFromKetValuePair(toolbox.AsString(key), value)
			if err != nil {
				err = e
				return false
			}
			result = append(result, variable)
			return true
		})
		return result, err
	}

	variables := toolbox.AsSlice(source)
	if len(variables) == 0 {
		return nil, nil
	}

	for _, item := range variables {
		switch value := item.(type) {
		case string:
			text := value
			if len(text) == 0 {
				continue
			}
			variableExpr := VariableExpression(toolbox.AsString(item))
			variable, err := variableExpr.AsVariable()
			if err != nil {
				return nil, err
			}
			result = append(result, variable)
		default:
			if toolbox.IsSlice(item) || toolbox.IsMap(item) {
				aMap, err := util.NormalizeMap(value, true)
				if err != nil {
					return nil, err
				}
				var variable = &Variable{}
				err = toolbox.DefaultConverter.AssignConverted(&variable, aMap)
				if err != nil {
					return nil, err
				}
				if variable.Name == "" && len(aMap) == 1 {
					for key, value := range aMap {
						variable, err = newVariableFromKetValuePair(toolbox.AsString(key), value)
						if err != nil {
							return nil, fmt.Errorf("unsupported variable definition: %v", value)
						}
					}
				}
				result = append(result, variable)
			} else {
				return nil, fmt.Errorf("unsupported type: %T", value)
			}
		}
	}
	return result, nil

}

func newVariableFromKetValuePair(key string, value interface{}) (*Variable, error) {
	var name = toolbox.AsString(key)
	isRequired := strings.HasPrefix(name, "!")
	if isRequired {
		name = string(name[1:])
	}
	if toolbox.IsSlice(value) {
		if normalized, err := util.NormalizeMap(value, false); err == nil {
			value = normalized
		}
		return &Variable{
			Name:              name,
			Value:             value,
			Required:          isRequired,
			EmptyIfUnexpanded: isRequired,
		}, nil
	} else {
		var variableExpr VariableExpression
		if strings.Contains(name, "=") {
			variableExpr = VariableExpression(fmt.Sprintf("%v: %v", name, value))
		} else {
			variableExpr = VariableExpression(fmt.Sprintf("%v = %v", name, value))
		}
		return variableExpr.AsVariable()
	}
}
