package iam

import (
	"github.com/viant/toolbox"
)

//Policy represent policy
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

//Principal represents policy principal
type Principal struct {
	Service string
}

//Statement represents policy document statement
type Statement struct {
	Sid       *string
	Effect    string
	Action    toolbox.AnyJSONType
	Resource  toolbox.AnyJSONType
	Principal toolbox.AnyJSONType
	Condition toolbox.AnyJSONType
}

//PolicyDocument represent policy document
type PolicyDocument struct {
	Version   string
	ID        string `json:"Id"`
	Statement []*Statement
}

type InfoStatement struct {
	SID       *string `yaml:"sid,omitempty"`
	Effect    string
	Action    interface{}
	Resource  interface{} `yaml:"resource,omitempty"`
	Condition interface{} `yaml:"condition,omitempty"`
	Principal interface{} `yaml:"principal,omitempty"`
}

type PolicyInfo struct {
	Statement []*InfoStatement
}
