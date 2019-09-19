package cloudwatchevents

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/cloudwatchevents"
	ciam "github.com/viant/endly/system/cloud/aws/iam"
	"github.com/aws/aws-sdk-go/service/iam"
)

//DeployRuleInput represents deploy rule input
type DeployRuleInput struct {
	cloudwatchevents.PutRuleInput
	ciam.SetupRolePolicyInput ` json:",inline"`
	Targets                   Targets
}

//DeployRuleOutput represents deployRule rule output
type DeployRuleOutput GetRuleOutput


//GetRuleInput represents get rule input
type GetRuleInput struct {
	Name *string
}

//GetRuleOutput represents get rule output
type GetRuleOutput struct {
	Rule     *cloudwatchevents.Rule
	Targets  []*cloudwatchevents.Target
	RoleInfo *ciam.GetRoleInfoOutput
}


//Init initialises request
func (i *DeployRuleInput) Init() error {
	if i.RoleName != nil {
		if i.DefaultPolicyDocument == nil {
			policyDocument := string(DefaultTrustRelationship)
			i.DefaultPolicyDocument = &policyDocument
		}
		if len(i.SetupRolePolicyInput.Define) == 0 {
			policyName := *i.RoleName + "-Role"
			policyDocument := string(DefaultRolePolicy)
			i.SetupRolePolicyInput.Define = make([]*iam.PutRolePolicyInput, 0)
			i.SetupRolePolicyInput.Define = append(i.SetupRolePolicyInput.Define, &iam.PutRolePolicyInput{
				PolicyName:     &policyName,
				PolicyDocument: &policyDocument,
			})
		}
	}
	return nil
}

//Validate checks if input is valid
func (i DeployRuleInput) Validate() error {
	if i.Name == nil {
		return fmt.Errorf("name was empty")
	}
	return nil
}


//DeleteRuleInput represents delete rule input
type DeleteRuleInput struct {
	cloudwatchevents.DeleteRuleInput
	TargetArn *string
}

func (i *DeleteRuleInput) Validate() error {
	if i.Name == nil && i.TargetArn == nil {
		return fmt.Errorf("name was empty")
	}
	return nil
}

