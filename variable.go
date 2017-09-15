package endly

import "github.com/viant/endly/common"

var VariableDefaultScope = "inout"
var VariableConstant = "const"

type Variable struct {
	Name   string
	Type   string //const,var
	Source string
	Scope  string //in, out, inout
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
	if v.in != nil {
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
		state.SetValue(variable.Name, Expand(state, variable.Source))
	}
}

func (v*Variables) Apply(in, out common.Map, inScope bool) {
	if v == nil || len(*v) == 0 {
		return
	}
	for _, variable := range *v {
		if inScope && ! variable.IsIn() {
			continue
		}
		if ! inScope && ! variable.IsOut() {
			continue
		}
		if variable.IsConstantType() {
			out.SetValue(variable.Name, Expand(in, variable.Source))
		}
		if value, has := in.GetValue(variable.Name); has {
			out.SetValue(variable.Name, value)
		}
	}
}
