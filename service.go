package endly

import (
	"github.com/viant/endly/common"
)


type Response struct {
	Status string
	Error error
	Response interface{}
}


type Service interface {
	Id() string
	State() common.Map
	Run(context *Context, request interface{}) *Response
	NewRequest(name string) (interface{}, error)
}



type AbstractService struct {
	Service
	id     string
	state  common.Map
}


func (s *AbstractService) Id() string {
	return s.id
}


func (s *AbstractService) State() common.Map {
	return s.state
}

func NewAbstractService(id string) *AbstractService {
	return &AbstractService{
		id:id,
		state:common.NewMap(),
	}
}