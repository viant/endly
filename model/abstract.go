package model

import "github.com/viant/endly/model/criteria/eval"

// AbstractNode represents an abstract workflow node(of a workflow, task or acton type)
type AbstractNode struct {
	Name        string
	Description string
	Init        Variables `description:"state init instruction "`
	Post        Variables `description:"post execution state update instruction"`
	When        string    `description:"run criteria"`
	whenEval   eval.Compute
	SleepTimeMs int   //optional Sleep time
	Logging     *bool `description:"optional flag to disable logging, enabled by default"`
}

func (n *AbstractNode) WhenEval() *eval.Compute {
	return &n.whenEval
}