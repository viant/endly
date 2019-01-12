package lambda

import "github.com/aws/aws-sdk-go/service/lambda"

//RecreateFunctionInput drops function if exist to create a new one
type RecreateFunctionInput lambda.CreateFunctionInput

//DropFunctionInput remove a function with all dependencies
type DropFunctionInput lambda.DeleteFunctionInput



//SetupFunctionInput setup function, creates or updates existing one
type SetupFunctionInput struct {

}


//SetupPermissionInput creates a permission if it does not exists
type SetupPermissionInput lambda.AddPermissionInput

