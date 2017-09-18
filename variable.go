package endly

import (
	"github.com/viant/endly/common"
	"strings"
)

var VariableDefaultScope = "inout"
var VariableConstant = "const"

type Variable struct {
	Name   string
	Type   string //const,var
	Source string
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
			out.SetValue(variable.Name, Expand(in, variable.Source))
		}

		if value, has := in.GetValue(variable.Source); has {
			out.SetValue(variable.Name, value)
		}
	}
}
