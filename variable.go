package endly

import (
	"github.com/viant/endly/common"
	"fmt"
)

var VariableDefaultScope = "inout"
var VariableConstant = "const"

type Variable struct {
	Name   string
	Type   string //const,var
	Source string
	Scope  string //init, in, out
	in     *bool
	out    *bool
}

func (v *Variable) IsConstantType() bool {
	return v.Type == VariableConstant
}

func (v *Variable) IsIn() bool {
	if v.in != nil {
		return *v.in
	}
	result := v.Scope == "" || v.Scope == VariableDefaultScope || v.Scope == "in"
	v.in = &result
	return *v.in
}

func (v *Variable) IsOut() bool {
	if v.out != nil {
		return *v.out
	}
	result := v.Scope == "" || v.Scope == VariableDefaultScope || v.Scope == "out"
	v.out = &result
	return *v.out
}

type Variables []*Variable

func (v*Variables) Eval(state common.Map) {
	if v == nil || len(*v) == 0 {
		return
	}
	for _, variable := range *v {
		value :=Expand(state, variable.Source)
		fmt.Printf("EVAL: %v=> %v\n", variable.Name, value)
		state.SetValue(variable.Name, value)
	}

	fmt.Printf("State: %v\n", state)
}

func (v*Variables) Apply(in, out common.Map, inScope bool) {

	if out == nil {
		return
	}

	if v == nil || len(*v) == 0 {
		return
	}
	for _, variable := range *v {

		if variable == nil {
			continue
		}
		if inScope && ! variable.IsIn() {
			continue
		}

		if ! inScope && ! variable.IsOut() {
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
