package endly

import (
	"github.com/viant/endly/common"
	"strings"
	"github.com/viant/toolbox"
)

var VariableDefaultScope = "inout"
var VariableConstant = "const"

type Variable struct {
	Name   string
	Type   string //const,var
	Source interface{}
	Scope  string //init, in, out
}

func (v *Variable) IsConstantType() bool {
	return v.Type == VariableConstant
}

type Variables []*Variable

func (v *Variables) Apply(in, out common.Map, scope string) {

	if out == nil {
		return
	}

	if v == nil || len(*v) == 0 {
		return
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
			if toolbox.IsMap(variable.Source){
				out.SetValue(variable.Name, expandMap(variable.Source, in))
			} else {
				out.SetValue(variable.Name, Expand(in, toolbox.AsString(variable.Source)))
			}
		}

		if value, has := in.GetValue(toolbox.AsString(variable.Source)); has {
			out.SetValue(variable.Name, value)
		}
	}
}
