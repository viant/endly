package model

import "github.com/viant/endly/util"

//Action represents a workflow service action
type Action struct {
	*AbstractNode
	*ServiceRequest
	*NeatlyTag
	*Repeater
	Async bool   `description:"flag to run action async"`
	Skip  string `description:"criteria to skip current TagID"`
}

//NewActivity returns pipeline activity
func (a *Action) Init() error {
	if a.AbstractNode == nil {
		a.AbstractNode = &AbstractNode{}
	}
	if a.ServiceRequest == nil {
		a.ServiceRequest = &ServiceRequest{}
	}
	if a.Repeater == nil {
		a.Repeater = &Repeater{}
	}
	if a.NeatlyTag == nil {
		a.NeatlyTag = &NeatlyTag{}
	}

	if a.Action != "" && a.Service == "" {
		selector := ActionSelector(a.Action)
		a.Service = selector.Service()
		a.Action = selector.Action()
	}
	if a.ServiceRequest != nil {
		a.ServiceRequest = a.ServiceRequest.Init()
	}
	a.Repeater = a.Repeater.Init()
	if err := a.Validate(); err != nil {
		return err
	}
	util.SetNonZero(&a.Repeater.SleepTimeMs, &a.AbstractNode.SleepTimeMs)
	util.SetNonEmpty(&a.ServiceRequest.Description, &a.ServiceRequest.Description)
	return nil
}

//ID returns action identified
func (a *Action) ID() string {
	if a.Name == "" {
		return a.Name
	}
	return a.Service + "_" + a.Action
}

//NeatlyTag represent a neatly tag
type NeatlyTag struct {
	Tag            string //neatly tag
	TagIndex       string //neatly tag index
	TagID          string //neatly tag id
	TagDescription string //tag description
}
