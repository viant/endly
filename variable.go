package endly

import (
	"bytes"
	"fmt"
	"github.com/viant/endly/common"
	"github.com/viant/toolbox"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

type Variable struct {
	Name     string
	Value    interface{}
	From     string
	Persist  bool //stores in tmp directory to be used as backup if data is not in the cotnext
	Required bool
}

func (v *Variable) tempfile() string {
	return path.Join(os.Getenv("TMPDIR"), v.Name+".var")
}

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

func (v *Variables) Apply(in, out common.Map) error {
	if v == nil || out == nil || in == nil || len(*v) == 0 {
		return nil
	}
	for _, variable := range *v {
		if variable == nil {
			continue
		}
		var value interface{}
		if variable.From != "" {
			udfFunction, fromKey, err := getUdfIfDefined(variable.From)
			if err != nil {
				return err
			}
			var has bool
			if udfFunction == nil {
				value, has = in.GetValue(variable.From)
				if !has {

					fromVariable := variable.fromVariable()
					err = fromVariable.Load()
					if fromVariable.Value != nil {
						in.SetValue(fromVariable.Name, fromVariable.Value)
						value, _ = in.GetValue(variable.From)
					}
					if err != nil {
						return err
					}
				}

			} else {
				value, _ = in.GetValue(fromKey)
				value, err = udfFunction(value, in)
				if err != nil {
					return err
				}
			}
		}

		//if variable.From != "" {
		//	udfFunction, fromKey, err := getUdfIfDefined(variable.From)
		//	if err != nil {
		//		return err
		//	}
		//	value, has := in.GetValue(fromKey)
		//	if ! has {
		//		fromVariable := variable.fromVariable()
		//		err = fromVariable.Load()
		//		fmt.Printf("VAL: %v %v %v\n", fromKey, fromVariable.Value, err)
		//		if err != nil {
		//			return err
		//		}
		//	}
		//	if udfFunction != nil {
		//		Value, err = udfFunction(Value, in)
		//		if err != nil {
		//			return err
		//		}
		//	}
		//}

		if value == nil {
			value = variable.Value
			if value != nil {
				value = ExpandValue(value, in)
			}
		}

		if variable.Required && (value == nil || toolbox.AsString(value) == "") {
			return fmt.Errorf("Variable %v is required, but was empty, %v", variable.Name, in)
		}
		out.SetValue(variable.Name, value)
	}
	return nil
}

func (v Variables) String() string {
	var result = ""
	for _, item := range v {
		result += fmt.Sprintf("{Name:%v From:%v Value:%v},", item.Name, item.From, item.Value)
	}
	return result
}
