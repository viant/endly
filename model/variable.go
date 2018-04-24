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
	Name     string            //name
	Value    interface{}       //default value
	From     string            //context state map key to pull data
	When     string            //criteria if specified this variable will be set only if evaluated criteria is true (it can use $in, and $out state variables)
	Else     interface{}       //if when criteria is not met then else can provide variable value alternative
	Persist  bool              //stores in tmp directory to be used as backup if data is not in the cotnext
	Required bool              //flag that validates that from returns non empty value or error is generated
	Replace  map[string]string `description:"replacements map, if key if specified substitute variable value with corresponding value. This will work only for string replacements"` //replacements map, if key if specified substitute variable value with corresponding value.
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

func (v *Variable) validate(value interface{}, in data.Map) error {
	if v.Required && (value == nil || toolbox.AsString(value) == "") {
		source := in.GetString(neatly.OwnerURL)
		return fmt.Errorf("variable '%v' is required by %v, but was empty, %v", v.Name, source, toolbox.MapKeysToStringSlice(in))
	}
	return nil
}

func (v *Variable) canApply(in, out data.Map) bool {
	var state data.Map = map[string]interface{}{
		"in":  in,
		"out": out,
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
func NewVariable(name, form, when string, required bool, value, elseValue interface{}, replace map[string]string) *Variable {
	return &Variable{
		Name:     name,
		From:     form,
		When:     when,
		Required: required,
		Value:    value,
		Else:     elseValue,
		Replace:  replace,
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

//VariableExpression represent a variable expression [!] [when  ?] VariableName = value : else, exclemation mark flags variable as required
type VariableExpression string

func normalizeValue(value string) interface{} {
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
		Required: isRequired,
	}
	var whenIndex = strings.Index(pair[0], "?")
	if whenIndex != -1 {
		result.When = string(pair[0][:whenIndex])
		result.Name = strings.TrimSpace(string(pair[0][whenIndex+1:]))
		elseIndex := strings.Index(pair[1], ":")
		if elseIndex != -1 {
			result.Value = string(pair[1][:elseIndex])
			result.Else = normalizeValue(string(pair[1][elseIndex+1:]))
		} else {
			result.Value = pair[1]
		}
	} else {
		result.Name = strings.TrimSpace(pair[0])
		result.Value = strings.TrimSpace(pair[1])
	}
	result.Value = normalizeValue(toolbox.AsString(result.Value))
	return result, nil
}

//GetVariables returns variables from Variables ([]*Variable), []string (as expression) or from []interface{} (where interface is a map matching Variable struct)
func GetVariables(baseURL string, source interface{}) (Variables, error) {
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
		err := util.Decode(baseURL, value, &result)
		return result, err
	}
	var result Variables = make([]*Variable, 0)
	if !toolbox.IsSlice(source) {
		return nil, fmt.Errorf("invalid varaibles type: %T, expected %T or %T", source, result, []string{})
	}

	if _, err := util.NormalizeMap(source, false); err == nil {
		toolbox.ProcessMap(source, func(key, value interface{}) bool {
			var name = toolbox.AsString(key)
			isRequired := strings.HasPrefix(name, "!")

			if isRequired {
				name = string(name[1:])
			}
			if toolbox.IsSlice(value) {
				if normalized, err := util.NormalizeMap(value, false); err == nil {
					value = normalized
				}
				result = append(result, &Variable{
					Name:  name,
					Value: value,
				})
			} else {
				result = append(result, &Variable{
					Name:  name,
					Value: value,
				})
			}

			return true
		})
		return result, nil
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
				result = append(result, variable)
			} else {
				return nil, fmt.Errorf("unsupported type: %T", value)
			}
		}
	}
	return result, nil

}
