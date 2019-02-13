package iam

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/iam"
)

//RecreateRoleInput drops role if exist to create a new one
type RecreateRoleInput iam.CreateRoleInput

//DropRoleInput deattaches all role polices and deletes role
type DropRoleInput iam.DeleteRoleInput

//SetupRolePolicyInput represents setup role policy input
type SetupRolePolicyInput struct {
	iam.CreateRoleInput
	DefaultPolicyDocument *string
	Attach                []*iam.AttachRolePolicyInput
	Define                []*iam.PutRolePolicyInput
}

//Validate checks if input is valid
func (i *SetupRolePolicyInput) Validate() error {
	if i.CreateRoleInput.RoleName == nil {
		return fmt.Errorf("roleName was empty")
	}
	return nil
}

type GetRoleInfoInput iam.GetRoleInput
type GetRoleInfoOutput struct {
	Role     *iam.Role
	Attached []*Policy
	Defined  []*Policy
}

type GetUserInfoInput iam.GetUserInput
type GetUserInfoOutput struct {
	User     *iam.User
	Attached []*Policy
	Defined  []*Policy
	Groups   []*GetGroupInfoOutput
}

type GetGroupInfoInput iam.GetGroupInput

type GetGroupInfoOutput struct {
	Group    *iam.Group
	Users    []*iam.User
	Attached []*Policy
	Defined  []*Policy
}
