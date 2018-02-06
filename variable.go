package endly

import (
	"bytes"
	"fmt"
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
	Name     string      //name
	Value    interface{} //default value
	From     string      //context state map key to pull data
	Persist  bool        //stores in tmp directory to be used as backup if data is not in the cotnext
	Required bool        //flag that validates that from returns non empty value or error is generated
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
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			return err
		}
		toolbox.NewJSONDecoderFactory().Create(bytes.NewReader(data)).Decode(&v.Value)
	}
	return nil
}

//Variables a slice of variables
type Variables []*Variable

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

func (v *Variables) getValueFromInput(variable *Variable, in data.Map) (interface{}, error) {
	var value interface{}
	if variable.From != "" {
		var has bool
		var key = variable.From
		if strings.HasPrefix(key, "!") {
			if !(strings.Contains(key, "[") || strings.Contains(key, "{")) {
				key = strings.Replace(key, "(", "($", 1)
			}
			value = in.Expand(key)
		} else {
			value, has = in.GetValue(key)
		}
		if !has {
			fromVariable := variable.fromVariable()
			err := fromVariable.Load()
			if fromVariable.Value != nil {
				in.SetValue(fromVariable.Name, fromVariable.Value)
				value, _ = in.GetValue(key)
			}
			if err != nil {
				return err, nil
			}
		}
	}
	return value, nil
}

func (v *Variables) validateRequiredValueIfNeeded(variable *Variable, value interface{}, in data.Map) error {
	if variable.Required && (value == nil || toolbox.AsString(value) == "") {
		source := in.GetString(neatly.OwnerURL)
		return fmt.Errorf("variable %v is required by %v, but was empty, %v", variable.Name, source, toolbox.MapKeysToStringSlice(in))
	}
	return nil
}

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
		value, err := v.getValueFromInput(variable, in)
		if err != nil {
			return err
		}
		if value == nil || (variable.Required && toolbox.AsString(value) == "") {
			value = variable.Value
			if value != nil {
				value = in.Expand(value)
			}
		}
		if err := v.validateRequiredValueIfNeeded(variable, value, in); err != nil {
			return err
		}

		if variable.Name != "" {
			out.SetValue(variable.Name, value)
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
