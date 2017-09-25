package common

import (
	"fmt"
	"github.com/viant/toolbox"
	"strings"
)

type Collection []interface{}

func (s *Collection) Push(value interface{}) {
	(*s) = append(*s, value)
}

func (s *Collection) ExpandWithMap(size int) {
	for i := len(*s); i < size; i++ {
		s.Push(NewMap())
	}
}

func (s *Collection) Range(handler func(item interface{}, index int) (bool, error)) error {
	for i, elem := range *s {
		next, err := handler(elem, i)
		if err != nil {
			return err
		}
		if !next {
			break
		}

	}
	return nil
}

func (s *Collection) RangeMap(handler func(item Map, index int) (bool, error)) error {
	var next bool
	var err error
	for i, elem := range *s {
		var aMap, ok = elem.(Map)
		if !ok {
			next, err = handler(nil, i)
		} else {
			next, err = handler(aMap, i)
		}
		if err != nil {
			return err
		}
		if !next {
			break
		}

	}
	return nil
}

func (s *Collection) String() string {

	fmt.Printf("S: %v\n", *s)
	var items = make([]string, 0)
	for _, item := range *s {
		items = append(items, toolbox.AsString(item))
	}
	return "[" + strings.Join(items, ",") + "]"
}

func (s *Collection) RangeString(handler func(item interface{}, index int) (bool, error)) error {
	for i, elem := range *s {
		next, err := handler(toolbox.AsString(elem), i)
		if err != nil {
			return err
		}
		if !next {
			break
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
		if !next {
			break
		}

	}
	return nil
}

func NewCollection() *Collection {
	var result Collection = make([]interface{}, 0)
	return &result
}
