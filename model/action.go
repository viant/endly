package model

// Action represents a workflow service action
type Action struct {
	*AbstractNode
	*ServiceRequest
	*MetaTag
	*Repeater
	Async bool   `description:"flag to run action async"`
	Skip  string `description:"criteria to skip current TagID"`
}

// NewActivity returns pipeline activity
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
	if a.MetaTag == nil {
		a.MetaTag = &MetaTag{}
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

	a.initSleepTime()
	return nil
}

/*
initSleepTimeMs initializes sleep time to be control by abstract node
*/
func (a *Action) initSleepTime() {
	if sleepTimeMs := a.Repeater.SleepTimeMs; sleepTimeMs > 0 {
		a.AbstractNode.SleepTimeMs = sleepTimeMs
		a.Repeater.SleepTimeMs = 0
	}
}

// Clone clones this actions
func (a *Action) Clone() *Action {
	abstract := *a.AbstractNode
	serviceRequest := *a.ServiceRequest
	metaTag := *a.MetaTag
	repeater := *a.Repeater
	return &Action{
		AbstractNode:   &abstract,
		ServiceRequest: &serviceRequest,
		MetaTag:        &metaTag,
		Repeater:       &repeater,
		Async:          a.Async,
		Skip:           a.Skip,
	}
}

// ID returns action identified
func (a *Action) ID() string {
	if a.Name == "" {
		return a.Name
	}
	return a.Service + "_" + a.Action
}

// MetaTag represent a node tag
type MetaTag struct {
	Tag            string //tag
	TagIndex       string //tag index
	TagID          string //tag id
	TagDescription string //tag description
	Comments       string
}
