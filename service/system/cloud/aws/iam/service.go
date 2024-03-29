package iam

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/viant/endly"
	"github.com/viant/endly/service/system/cloud/aws"
	"log"
	"time"
)

const (
	//ServiceID aws iam service id.
	ServiceID = "aws/iam"
)

var permissionPropagation = 5 * time.Second

// no operation service
type service struct {
	*endly.AbstractService
}

func (s *service) dropRole(context *endly.Context, request *DropRoleInput) (interface{}, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	policies, err := client.ListRolePolicies(&iam.ListRolePoliciesInput{
		RoleName: request.RoleName,
	})
	if err != nil {
		return nil, err
	}
	if policies != nil && len(policies.PolicyNames) > 0 {
		for _, policy := range policies.PolicyNames {
			if _, err = client.DeleteRolePolicy(&iam.DeleteRolePolicyInput{
				PolicyName: policy,
				RoleName:   request.RoleName,
			}); err != nil {
				return nil, fmt.Errorf("failed to delete policy %v", err)
			}
		}
	}
	output, err := client.ListAttachedRolePolicies(&iam.ListAttachedRolePoliciesInput{
		RoleName: request.RoleName,
	})
	if err != nil {
		return nil, err
	}
	if output != nil && len(output.AttachedPolicies) > 0 {
		for _, attached := range output.AttachedPolicies {
			if _, err = client.DetachRolePolicy(&iam.DetachRolePolicyInput{
				PolicyArn: attached.PolicyArn,
				RoleName:  request.RoleName,
			}); err != nil {
				return nil, fmt.Errorf("failed to detach policy %v", err)
			}
		}
	}
	return client.DeleteRole(&iam.DeleteRoleInput{
		RoleName: request.RoleName,
	})
}

func (s *service) recreateRole(context *endly.Context, request *RecreateRoleInput) (*iam.CreateRoleOutput, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	output, err := client.GetRole(&iam.GetRoleInput{
		RoleName: request.RoleName,
	})
	if err == nil && output.Role != nil {
		if _, err = s.dropRole(context, &DropRoleInput{
			RoleName: request.RoleName,
		}); err != nil {
			return nil, err
		}
	}
	state := context.State()
	if request.AssumeRolePolicyDocument != nil {
		*request.AssumeRolePolicyDocument = state.ExpandAsText(*request.AssumeRolePolicyDocument)
	}
	iamRequest := iam.CreateRoleInput(*request)
	return client.CreateRole(&iamRequest)
}

func (s *service) setupRole(context *endly.Context, request *SetupRolePolicyInput) (*GetRoleInfoOutput, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	roleOutput, foundErr := client.GetRole(&iam.GetRoleInput{
		RoleName: request.RoleName,
	})

	state := context.State()
	if request.AssumeRolePolicyDocument != nil {
		*request.AssumeRolePolicyDocument = state.ExpandAsText(*request.AssumeRolePolicyDocument)
	}

	var role *iam.Role
	if foundErr != nil {
		_, err := client.CreateRole(&request.CreateRoleInput)
		if err != nil {
			return nil, err
		}
		//wait for role to propagate
		time.Sleep(permissionPropagation)
		roleOutput, _ = client.GetRole(&iam.GetRoleInput{
			RoleName: request.RoleName,
		})
	}
	role = roleOutput.Role
	if request.AssumeRolePolicyDocument != nil {
		if _, err = client.UpdateAssumeRolePolicy(&iam.UpdateAssumeRolePolicyInput{
			PolicyDocument: request.AssumeRolePolicyDocument,
			RoleName:       request.RoleName,
		}); err != nil {
			return nil, err
		}
	}
	attached := false
	if attached, err = s.setupAttachedRolePolicy(context, role, request); err != nil {
		return nil, err
	}

	updated := false
	if updated, err = s.setupRolePolicy(context, role, request); err != nil {
		return nil, err
	}
	if attached || updated {
		time.Sleep(permissionPropagation)
	}
	return s.getRoleInfo(context, &GetRoleInfoInput{
		RoleName: request.RoleName,
	})
}

func (s *service) setupRolePolicy(context *endly.Context, role *iam.Role, request *SetupRolePolicyInput) (bool, error) {
	client, err := GetClient(context)
	if err != nil {
		return false, err
	}
	state := context.State()
	if len(request.Define) == 0 {
		return false, err
	}
	var alreadyDefined = make(map[string]bool)
	outputList, _ := client.ListRolePolicies(&iam.ListRolePoliciesInput{
		RoleName: role.RoleName,
	})
	if outputList != nil && len(outputList.PolicyNames) > 0 {
		for _, policy := range outputList.PolicyNames {
			alreadyDefined[*policy] = true
		}
	}

	updated := 0
	for _, define := range request.Define {
		*define.PolicyDocument = state.ExpandAsText(*define.PolicyDocument)

		if _, has := alreadyDefined[*define.PolicyName]; has {
			delete(alreadyDefined, *define.PolicyName)
			policy, _ := client.GetRolePolicy(&iam.GetRolePolicyInput{
				PolicyName: define.PolicyName,
				RoleName:   role.RoleName,
			})
			if *policy.PolicyDocument == *define.PolicyDocument {
				continue
			}
			if _, err = client.DeleteRolePolicy(&iam.DeleteRolePolicyInput{
				PolicyName: define.PolicyName,
				RoleName:   role.RoleName,
			}); err != nil {
				return false, err
			}
		}
		define.RoleName = role.RoleName
		updated++
		if _, err = client.PutRolePolicy(define); err != nil {
			return false, err
		}
	}
	for policyName := range alreadyDefined {
		if _, err = client.DeleteRolePolicy(&iam.DeleteRolePolicyInput{
			PolicyName: &policyName,
			RoleName:   role.RoleName,
		}); err != nil {
			return false, err
		}
	}
	return updated > 0, err
}

func (s *service) setupAttachedRolePolicy(context *endly.Context, role *iam.Role, request *SetupRolePolicyInput) (bool, error) {
	client, err := GetClient(context)
	if err != nil {
		return false, err
	}
	if len(request.Attach) == 0 {
		return false, err
	}
	if role == nil {
		return false, fmt.Errorf("role was empty %v", request.RoleName)
	}
	attachedCount := 0
	var alreadyAttached = make(map[string]bool)
	attachedOutput, _ := client.ListAttachedRolePolicies(&iam.ListAttachedRolePoliciesInput{RoleName: role.RoleName})
	if attachedOutput != nil && len(attachedOutput.AttachedPolicies) > 0 {
		for _, attached := range attachedOutput.AttachedPolicies {
			alreadyAttached[*attached.PolicyArn] = true
		}
	}
	for _, attach := range request.Attach {
		if _, has := alreadyAttached[*attach.PolicyArn]; has {
			delete(alreadyAttached, *attach.PolicyArn)
			continue
		}
		attachedCount++
		if _, err = client.AttachRolePolicy(&iam.AttachRolePolicyInput{
			RoleName:  role.RoleName,
			PolicyArn: attach.PolicyArn,
		}); err != nil {
			return false, err
		}
	}
	for arn := range alreadyAttached {
		if _, err = client.DetachRolePolicy(&iam.DetachRolePolicyInput{
			PolicyArn: &arn,
			RoleName:  role.RoleName,
		}); err != nil {
			return false, err
		}
	}
	return attachedCount > 0, err
}

func (s *service) getPolicyVersion(context *endly.Context, policyArn string) (*iam.PolicyVersion, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	policyOutput, err := client.GetPolicy(&iam.GetPolicyInput{
		PolicyArn: &policyArn,
	})
	policyVersion, err := client.GetPolicyVersion(&iam.GetPolicyVersionInput{
		PolicyArn: &policyArn,
		VersionId: policyOutput.Policy.DefaultVersionId,
	})
	if err != nil {
		return nil, err
	}
	return policyVersion.PolicyVersion, nil
}

func (s *service) getRoleInfo(context *endly.Context, request *GetRoleInfoInput) (*GetRoleInfoOutput, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	output := &GetRoleInfoOutput{
		Defined:  make([]*Policy, 0),
		Attached: make([]*Policy, 0),
	}
	roleOutput, err := client.GetRole(&iam.GetRoleInput{
		RoleName: request.RoleName,
	})
	if err != nil {
		return nil, err
	}
	output.Role = roleOutput.Role
	if policies, _ := client.ListRolePolicies(&iam.ListRolePoliciesInput{RoleName: request.RoleName}); policies != nil && len(policies.PolicyNames) > 0 {
		for _, name := range policies.PolicyNames {
			if policyOutput, err := client.GetRolePolicy(&iam.GetRolePolicyInput{PolicyName: name, RoleName: request.RoleName}); err == nil {
				policy := &Policy{
					PolicyName: policyOutput.PolicyName,
					Document:   policyOutput.PolicyDocument,
				}
				if policy.PolicyDocument, err = getPolicyDocument(*policyOutput.PolicyDocument); err != nil {
					return nil, err
				}
				output.Defined = append(output.Defined, policy)
			}
		}
	}

	if attachedOutput, err := client.ListAttachedRolePolicies(&iam.ListAttachedRolePoliciesInput{RoleName: request.RoleName}); attachedOutput != nil && len(attachedOutput.AttachedPolicies) > 0 {
		for _, attached := range attachedOutput.AttachedPolicies {
			policy := &Policy{
				PolicyName: attached.PolicyName,
				PolicyArn:  attached.PolicyArn,
			}
			if version, e := s.getPolicyVersion(context, *attached.PolicyArn); e == nil {
				if policy.PolicyDocument, err = getPolicyDocument(*version.Document); err != nil {
					return nil, err
				}
			}
			output.Attached = append(output.Attached, policy)
		}
	}
	return output, nil
}

func (s *service) getGroupInfo(context *endly.Context, request *GetGroupInfoInput) (*GetGroupInfoOutput, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	output := &GetGroupInfoOutput{
		Defined: make([]*Policy, 0),
	}
	groupOutput, err := client.GetGroup(&iam.GetGroupInput{GroupName: request.GroupName})
	if err != nil {
		return nil, err
	}
	groupOutput.Users = groupOutput.Users
	output.Group = groupOutput.Group
	if policies, _ := client.ListGroupPolicies(&iam.ListGroupPoliciesInput{GroupName: request.GroupName}); policies != nil && len(policies.PolicyNames) > 0 {
		for _, name := range policies.PolicyNames {

			if policyOutput, err := client.GetGroupPolicy(&iam.GetGroupPolicyInput{PolicyName: name, GroupName: request.GroupName}); err == nil {
				policy := &Policy{
					PolicyName: policyOutput.PolicyName,
					Document:   policyOutput.PolicyDocument,
				}
				if policy.PolicyDocument, err = getPolicyDocument(*policyOutput.PolicyDocument); err != nil {
					return nil, err
				}
				output.Defined = append(output.Defined, policy)
			}
		}
	}

	if attachedOutput, err := client.ListAttachedGroupPolicies(&iam.ListAttachedGroupPoliciesInput{GroupName: request.GroupName}); attachedOutput != nil && len(attachedOutput.AttachedPolicies) > 0 {
		for _, attached := range attachedOutput.AttachedPolicies {
			policy := &Policy{
				PolicyName: attached.PolicyName,
				PolicyArn:  attached.PolicyArn,
			}
			if version, e := s.getPolicyVersion(context, *attached.PolicyArn); e == nil {
				if policy.PolicyDocument, err = getPolicyDocument(*version.Document); err != nil {
					return nil, err
				}
			}
			output.Attached = append(output.Attached, policy)
		}
	}
	return output, nil
}

func (s *service) getUserInfo(context *endly.Context, request *GetUserInfoInput) (*GetUserInfoOutput, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	output := &GetUserInfoOutput{
		Defined:  make([]*Policy, 0),
		Attached: make([]*Policy, 0),
		Groups:   make([]*GetGroupInfoOutput, 0),
	}
	userOutput, err := client.GetUser(&iam.GetUserInput{UserName: request.UserName})
	if err != nil {
		return nil, err
	}
	output.User = userOutput.User
	if policies, _ := client.ListUserPolicies(&iam.ListUserPoliciesInput{UserName: request.UserName}); policies != nil && len(policies.PolicyNames) > 0 {
		for _, name := range policies.PolicyNames {
			if policyOutput, err := client.GetUserPolicy(&iam.GetUserPolicyInput{PolicyName: name, UserName: request.UserName}); err == nil {
				policy := &Policy{
					PolicyName: policyOutput.PolicyName,
					Document:   policyOutput.PolicyDocument,
				}
				if policy.PolicyDocument, err = getPolicyDocument(*policyOutput.PolicyDocument); err != nil {
					return nil, err
				}
				output.Defined = append(output.Defined, policy)
			}
		}
	}
	if attachedOutput, err := client.ListAttachedUserPolicies(&iam.ListAttachedUserPoliciesInput{UserName: request.UserName}); attachedOutput != nil && len(attachedOutput.AttachedPolicies) > 0 {
		for _, attached := range attachedOutput.AttachedPolicies {
			policy := &Policy{
				PolicyName: attached.PolicyName,
				PolicyArn:  attached.PolicyArn,
			}
			if version, e := s.getPolicyVersion(context, *attached.PolicyArn); e == nil {
				if policy.PolicyDocument, err = getPolicyDocument(*version.Document); err != nil {
					return nil, err
				}
			}
			output.Attached = append(output.Attached, policy)
		}
	}
	if groupsOutput, _ := client.ListGroupsForUser(&iam.ListGroupsForUserInput{
		UserName: request.UserName,
	}); groupsOutput != nil && len(groupsOutput.Groups) > 0 {
		for _, group := range groupsOutput.Groups {
			groupInfo, err := s.getGroupInfo(context, &GetGroupInfoInput{GroupName: group.GroupName})
			if err != nil {
				return nil, err
			}
			output.Groups = append(output.Groups, groupInfo)
		}
	}
	return output, nil
}

func (s *service) registerRoutes() {
	client := &iam.IAM{}
	routes, err := aws.BuildRoutes(client, getClient)
	if err != nil {
		log.Printf("unable register service %v actions: %v\n", ServiceID, err)
		return
	}
	for _, route := range routes {
		route.OnRawRequest = setClient
		s.Register(route)
	}
	s.Register(&endly.Route{
		Action: "recreateRole",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "recreateRole", &RecreateRoleInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &iam.CreateRoleOutput{}),
		},
		RequestProvider: func() interface{} {
			return &RecreateRoleInput{}
		},
		ResponseProvider: func() interface{} {
			return &iam.CreateRoleOutput{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*RecreateRoleInput); ok {
				return s.recreateRole(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "setupRole",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "setupRole", &SetupRolePolicyInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &GetRoleInfoOutput{}),
		},
		RequestProvider: func() interface{} {
			return &SetupRolePolicyInput{}
		},
		ResponseProvider: func() interface{} {
			return &GetRoleInfoOutput{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*SetupRolePolicyInput); ok {
				return s.setupRole(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "getRoleInfo",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "getRoleInfo", &GetRoleInfoInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &iam.CreateRoleOutput{}),
		},
		RequestProvider: func() interface{} {
			return &GetRoleInfoInput{}
		},
		ResponseProvider: func() interface{} {
			return &GetRoleInfoOutput{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*GetRoleInfoInput); ok {
				return s.getRoleInfo(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "getUserInfo",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "setUserInfo", &GetUserInfoInput{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &GetUserInfoOutput{}),
		},
		RequestProvider: func() interface{} {
			return &GetUserInfoInput{}
		},
		ResponseProvider: func() interface{} {
			return &GetUserInfoOutput{}
		},
		OnRawRequest: setClient,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*GetUserInfoInput); ok {
				return s.getUserInfo(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

// New creates a new AWS IAM service.
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
