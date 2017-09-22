package endly

import (
	"github.com/viant/endly/common"
	"github.com/viant/toolbox"
	"strings"
)

var VariableDefaultScope = "inout"
var VariableConstant = "const"

type Variable struct {
	Name    string
	Type    string //const,var
	Source  interface{}
	Default string
	Scope   string //init, in, out
}

func (v *Variable) IsConstantType() bool {
	return v.Type == VariableConstant
}

type Variables []*Variable

func (v *Variables) Apply(in, out common.Map, scope string) error {

	if out == nil {
		return nil
	}

	if v == nil || len(*v) == 0 {
		return nil
	}
	scope = strings.ToLower(scope)
	for _, variable := range *v {

		if variable == nil {
			continue
		}
		if variable.Scope != "" && strings.ToLower(variable.Scope) != scope {
			continue
		}
		if variable.IsConstantType() {
			if toolbox.IsMap(variable.Source) {

				aMap, err := ExpandAsMap(variable.Source, in)
				if err != nil {
					return err
				}
				out.SetValue(variable.Name, aMap)
			} else {
				var value = Expand(in, toolbox.AsString(variable.Source))
				if value == "" {
					value = variable.Default
				}
				out.SetValue(variable.Name, Expand(in, toolbox.AsString(variable.Source)))
			}
		}

		if value, has := in.GetValue(toolbox.AsString(variable.Source)); has {
			out.SetValue(variable.Name, value)
		}
	}
	return nil
}
