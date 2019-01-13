package lambda

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/lambda"
	ciam "github.com/viant/endly/system/cloud/aws/iam"
)

//RecreateFunctionInput drops function if exist to create a new one
type RecreateFunctionInput lambda.CreateFunctionInput

//DropFunctionInput remove a function with all dependencies
type DropFunctionInput lambda.DeleteFunctionInput



//SetupFunctionInput setup function, creates or updates existing one
type SetupFunctionInput struct {
	*lambda.CreateFunctionInput
	*ciam.SetupRolePolicyInput
}




func (i *SetupFunctionInput) Init() error {
	if i.SetupRolePolicyInput == nil {
		return nil
	}

	if i.DefaultPolicyDocument == nil {
		policyDocument := string(DefaultTrustPolicy)
		i.DefaultPolicyDocument = &policyDocument
	}

	return nil
}

func (i *SetupFunctionInput) Validate() error {
	if i.CreateFunctionInput == nil {
		return fmt.Errorf("functionName was empty")
	}
	if i.CreateFunctionInput.Code == nil {
		return fmt.Errorf("code was empty")
	}
	if i.SetupRolePolicyInput == nil {
		return fmt.Errorf("roleName was empty")
	}

	return nil
}



type SetupFunctionOutput struct {
	*lambda.FunctionConfiguration
	RoleInfo *ciam.GetRoleInfoOutput
}


//SetupPermissionInput creates a permission if it does not exists
type SetupPermissionInput lambda.AddPermissionInput

