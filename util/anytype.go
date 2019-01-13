package util

import (
	"encoding/json"
)

type AnyType string

func (s *AnyType) UnmarshalJSON(b []byte) error {
	*s = AnyType(b)
	return nil
}

func (s *AnyType) MarshalJSON() ([]byte, error) {
	return []byte(*s), nil
}

//Value returns string or string slice value
func (s AnyType) Value() (interface{}, error) {
	var result  interface{}
	return result, json.Unmarshal([]byte(s), &result)
}
