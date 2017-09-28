package endly

import (
	"github.com/viant/endly/common"
	"fmt"
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
			udfFunction, fromKey, err := applyUdf(variable.From)
			if err != nil {
				return err
			}
			if udfFunction == nil {
				value, _ = in.GetValue(variable.From)
			} else {
				value, _ = in.GetValue(fromKey)
				value, err = udfFunction(value, in)
				if err != nil {
					return err
				}
			}
		}

		if value == nil {
			value = variable.Value
			value = ExpandValue(value, in)
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
