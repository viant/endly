package iam

import (
	"github.com/viant/toolbox"
)

// Policy represent policy
type Policy struct {
	PolicyName     *string
	PolicyArn      *string
	Document       *string
	PolicyDocument *PolicyDocument
}

func (p *Policy) PolicyInfo() []*InfoStatement {
	result := &PolicyInfo{
		Statement: make([]*InfoStatement, 0),
	}
	if p.PolicyDocument != nil && len(p.PolicyDocument.Statement) > 0 {
		for _, statement := range p.PolicyDocument.Statement {
			info := &InfoStatement{
				SID:    statement.Sid,
				Effect: statement.Effect,
			}
			info.Action, _ = statement.Action.Value()
			info.Resource, _ = statement.Resource.Value()
			info.Condition, _ = statement.Condition.Value()
			info.Principal, _ = statement.Principal.Value()
			result.Statement = append(result.Statement, info)
		}
	}
	return result.Statement
}

// Principal represents policy principal
type Principal struct {
	Service string
}

// Statement represents policy document statement
type Statement struct {
	Sid       *string
	Effect    string
	Action    toolbox.AnyJSONType `json:",omitempty"`
	Resource  toolbox.AnyJSONType `json:",omitempty"`
	Principal toolbox.AnyJSONType `json:",omitempty"`
	Condition toolbox.AnyJSONType `json:",omitempty"`
}

// PolicyDocument represent policy document
type PolicyDocument struct {
	Version   string
	ID        string `json:"Id"`
	Statement []*Statement
}

type InfoStatement struct {
	SID       *string `yaml:"sid,omitempty" json:",omitempty"`
	Effect    string
	Action    interface{}
	Resource  interface{} `yaml:"resource,omitempty"  json:",omitempty"`
	Condition interface{} `yaml:"condition,omitempty"  json:",omitempty"`
	Principal interface{} `yaml:"principal,omitempty"  json:",omitempty"`
}

type PolicyInfo struct {
	Statement []*InfoStatement
}
