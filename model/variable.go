package model

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/viant/neatly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/endly/criteria"
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
	if v.Value == nil {
		var filename = v.tempfile()
		if !toolbox.FileExists(filename) {
			return nil
		}
		filedata, err := ioutil.ReadFile(filename)
		if err != nil {
			return err
		}
		toolbox.NewJSONDecoderFactory().Create(bytes.NewReader(filedata)).Decode(&v.Value)
	}
	return nil
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
			value = in.Expand(from)
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

func (v *Variable) applyElse(in, out data.Map) {
	value := v.getElse(in)
	if v.Name != "" {
		out.SetValue(v.Name, value)
	}
}

func (v *Variable) applyDefault(in, out data.Map) {
	value := v.getValue(in)
	if v.Name != "" {
		out.SetValue(v.Name, value)
	}
}

func (v *Variable) Apply(in, out data.Map) error {
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

//Variables a slice of variables
type Variables []*Variable

func (v *Variables) isContextEmpty(in, out data.Map) bool {
	if v == nil || out == nil || in == nil || len(*v) == 0 {
		return true
	}
	return false
}

//Apply evaluates all variable from in map to out map
func (v *Variables) Apply(in, out data.Map) error {
	if v.isContextEmpty(in, out) {
		return nil
	}
	for _, variable := range *v {
		if variable == nil {
			continue
		}

		if variable.When != "" {
			if !variable.canApply(in, out) {
				variable.applyElse(in, out)
				continue
			}
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
		result += fmt.Sprintf("{Name:%v From:%v Value:%v},", item.Name, item.From, item.Value)
	}
	return result
}



//ModifiedStateEvent represent modified state event
type ModifiedStateEvent struct {
	Variables Variables
	In        map[string]interface{}
	Modified  map[string]interface{}
}


//NewModifiedStateEvent creates a new modified state event.
func NewModifiedStateEvent(variables Variables, in, out data.Map) *ModifiedStateEvent {
	var result = &ModifiedStateEvent{
		Variables: variables,
		In:        make(map[string]interface{}),
		Modified:  make(map[string]interface{}),
	}
	for _, variable := range variables {
		from := data.ExtractPath(variable.From)
		result.In[from], _ = in.GetValue(from)
		name := data.ExtractPath(variable.Name)
		result.Modified[name], _ = out.GetValue(name)
	}
	return result
}
