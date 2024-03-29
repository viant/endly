package model

import "github.com/viant/endly/model/criteria/eval"

// AbstractNode represents an abstract workflow node(of a workflow, task or acton type)
type AbstractNode struct {
	Name        string
	Description string    `yaml:",omitempty"`
	Init        Variables `description:"state init instruction " yaml:",omitempty"`
	Post        Variables `description:"post execution state update instruction" yaml:",omitempty"`
	When        string    `description:"run criteria" yaml:",omitempty"`
	SleepTimeMs int       `yaml:",omitempty"`
	Logging     *bool     `description:"optional flag to disable logging, enabled by default" yaml:",omitempty"`
	whenEval    eval.Compute
}

func (n *AbstractNode) WhenEval() *eval.Compute {
	return &n.whenEval
}

func (n *AbstractNode) Clone() *AbstractNode {
	ret := *n
	return &ret
}
