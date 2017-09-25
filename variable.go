package endly

import (
	"fmt"
	"github.com/viant/endly/common"
)

type Variable struct {
	Name  string
	Value interface{}
	From  string
}

type Variables []*Variable

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
			fmt.Printf("FORM :%v\n", variable.From)
			value, _ = in.GetValue(variable.From)
		}
		if value == nil {
			value = variable.Value
		}
		value = ExpandValue(value, in)
		out.SetValue(variable.Name, value)
	}
	return nil
}
