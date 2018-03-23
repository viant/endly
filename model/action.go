package model


//Action represents a workflow service action
type Action struct {
	*ServiceRequest
	*NeatlyTag
	*Repeater
	Name  string    `description:"action name"`
	Init  Variables `description:"action state init instruction "`
	Post  Variables `description:"action post processing instruction"`
	Async bool      `description:"flag to run action async"`
	When  string    `description:"run action criteria"`
	Skip  string    `description:"criteria to skip current TagID"`
}


//NeatlyTag represent a neatly tag
type NeatlyTag struct {
	Tag            string //neatly tag
	TagIndex       string //neatly tag index
	TagID          string //neatly tag id
	TagDescription string //tag description
}

