package iam

import (
	"github.com/viant/endly/model/msg"
	"gopkg.in/yaml.v2"
)

type PolicyEvenInfo struct {
	Policy   *string          `yaml:"policy,omitempty"`
	Arn      *string          `yaml:"arn,omitempty"`
	Document []*InfoStatement `yaml:"document,omitempty"`
}

type RoleEventInfo struct {
	Role     string
	Arn      string
	Defined  []*PolicyEvenInfo `yaml:"defined,omitempty"`
	Attached []*PolicyEvenInfo `yaml:"attached,omitempty"`
}

func (e *RoleEventInfo) Messages() []*msg.Message {
	info := ""
	if content, err := yaml.Marshal(e); err == nil {
		info = string(content)
	}
	return []*msg.Message{
		msg.NewMessage(msg.NewStyled(e.Role, msg.MessageStyleGeneric),
			msg.NewStyled("role", msg.MessageStyleGeneric),
			msg.NewStyled(info, msg.MessageStyleOutput),
		),
	}
}

func NewRoleEventInfo(output *GetRoleInfoOutput) *RoleEventInfo {

	return &RoleEventInfo{
		Role:     *output.Role.RoleName,
		Arn:      *output.Role.Arn,
		Attached: getPolicies(output.Attached),
		Defined:  getPolicies(output.Defined),
	}
}

type GroupEventInfo struct {
	Group    *string
	Arn      *string
	Users    []string          `yaml:"users,omitempty"`
	Defined  []*PolicyEvenInfo `yaml:"defined,omitempty"`
	Attached []*PolicyEvenInfo `yaml:"attached,omitempty"`
}

func (e *GroupEventInfo) Messages() []*msg.Message {
	info := ""
	if content, err := yaml.Marshal(e); err == nil {
		info = string(content)
	}
	return []*msg.Message{
		msg.NewMessage(msg.NewStyled(*e.Group, msg.MessageStyleGeneric),
			msg.NewStyled("group", msg.MessageStyleGeneric),
			msg.NewStyled(info, msg.MessageStyleOutput),
		),
	}
}

func NewGroupEventInfo(output *GetGroupInfoOutput, includeUsers bool) *GroupEventInfo {
	var result = &GroupEventInfo{
		Group:    output.Group.GroupName,
		Arn:      output.Group.Arn,
		Defined:  getPolicies(output.Defined),
		Attached: getPolicies(output.Attached),
		Users:    make([]string, 0),
	}
	if includeUsers && len(output.Users) > 0 {
		for _, user := range output.Users {
			result.Users = append(result.Users, *user.UserName)
		}
	}
	return result
}

type UserEventInfo struct {
	User     *string
	Arn      *string
	Defined  []*PolicyEvenInfo `yaml:"defined,omitempty"`
	Attached []*PolicyEvenInfo `yaml:"attached,omitempty"`
	Groups   []*GroupEventInfo `yaml:"groups,omitempty"`
}

func (e *UserEventInfo) Messages() []*msg.Message {
	info := ""
	if content, err := yaml.Marshal(e); err == nil {
		info = string(content)
	}
	return []*msg.Message{
		msg.NewMessage(msg.NewStyled(*e.User, msg.MessageStyleGeneric),
			msg.NewStyled("user", msg.MessageStyleGeneric),
			msg.NewStyled(info, msg.MessageStyleOutput),
		),
	}
}

func NewUserEventInfo(output *GetUserInfoOutput) *UserEventInfo {
	result := &UserEventInfo{
		User:     output.User.UserName,
		Arn:      output.User.Arn,
		Attached: getPolicies(output.Attached),
		Defined:  getPolicies(output.Defined),
		Groups:   make([]*GroupEventInfo, 0),
	}
	if len(output.Groups) > 0 {
		for _, groupOutput := range output.Groups {
			result.Groups = append(result.Groups, NewGroupEventInfo(groupOutput, false))
		}
	}
	return result
}

func (o *GetRoleInfoOutput) Messages() []*msg.Message {
	if o == nil {
		return nil
	}
	event := NewRoleEventInfo(o)
	return event.Messages()
}

func (o *GetUserInfoOutput) Messages() []*msg.Message {
	if o == nil {
		return nil
	}
	event := NewUserEventInfo(o)
	return event.Messages()
}
