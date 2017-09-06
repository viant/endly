package common

import "github.com/viant/toolbox"

type Collection []interface{}

func (s *Collection) Push(value interface{}) {
	(*s) = append(*s, value)
}



func (s *Collection) Range(handler func(item interface{}, index int) (bool, error)) error {
	for i, elem := range *s {
		next, err := handler(elem, i)
		if err != nil {
			return err
		}
		if ! next {
			break;
		}

	}
	return nil
}

func (s *Collection) RangeString(handler func(item interface{}, index int) (bool, error)) error {
	for i, elem := range *s {
		next, err := handler(toolbox.AsString(elem), i)
		if err != nil {
			return err
		}
		if ! next {
			break;
		}

	}
	return nil

}


func (s *Collection) RangeInt(handler func(item interface{}, index int) (bool, error)) error {
	for i, elem := range *s {
		next, err := handler(toolbox.AsInt(elem), i)
		if err != nil {
			return err
		}
		if ! next {
			break;
		}

	}
	return nil
}

func NewCollection() *Collection {
	var result Collection = make([]interface{}, 0)
	return &result
}