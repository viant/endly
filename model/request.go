package model

import (
	"errors"
	"fmt"
)

const defaultServiceID = "workflow"

var emptyRequest = map[string]interface{}{}

//ServiceRequest represent an action request
type ServiceRequest struct {
	Service     string      `description:"service ID"`
	Action      string      `description:"service's'action "`
	Request     interface{} `description:"service request"`
}

//Init initialises action
func (r *ServiceRequest) Init() *ServiceRequest {
	if r == nil {
		r = &ServiceRequest{}
	}
	if r.Service == "" {
		r.Service = defaultServiceID
	}
	if r.Request == nil {
		r.Request = emptyRequest
	}
	return r
}

//Validate check is action request is valid
func (r *ServiceRequest) Validate() error {
	if r.Service == "" {
		return errors.New("service  was empty")
	}
	if r.Action == "" {
		return errors.New("action was empty")
	}
	if r.Request == nil {
		return fmt.Errorf("request was nil for %v.%v", r.Service, r.Action)
	}
	return nil
}

func (r *ServiceRequest) NewAction() *Action {
	var repeater = &Repeater{}
	return &Action{
		AbstractNode: &AbstractNode{
		},
		ServiceRequest: r,
		MetaTag:        &MetaTag{},
		Repeater:       repeater.Init(),
	}
}
