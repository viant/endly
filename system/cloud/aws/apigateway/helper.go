package apigateway

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/viant/endly"
	"github.com/viant/toolbox/data"
)

func indexResource(resources []*apigateway.Resource) map[string]*apigateway.Resource {
	var result = make(map[string]*apigateway.Resource)
	if len(resources) == 0 {
		return result
	}
	for _, resource := range resources {
		result[*resource.Path] = resource
	}
	return result
}

/*
 { "op": "add", "path": "/a/b/c", "value": "foo" }
   { "path": "/a/b/c", "op": "add", "value": "foo" }
   { "value": "foo", "path": "/a/b/c", "op": "add" }

 */

func patchBool(source, target *bool, path string) (*apigateway.PatchOperation, bool) {
	if source == nil {
		if target == nil {
			return nil, false
		}
		return &apigateway.PatchOperation{
			Op:    aws.String("add"),
			Path:  aws.String(path),
			Value: aws.String(fmt.Sprintf("%v", *target)),
		}, true
	}
	if target == nil {
		return &apigateway.PatchOperation{
			Op:   aws.String("remove"),
			Path: aws.String(path),
			From: aws.String(fmt.Sprintf("%v", *source)),
		}, true
	}
	if *target == *source {
		return nil, false
	}
	return &apigateway.PatchOperation{
		Op:    aws.String("replace"),
		Path:  aws.String(path),
		From:  aws.String(fmt.Sprintf("%v", *source)),
		Value: aws.String(fmt.Sprintf("%v", *target)),
	}, true
}

func patchString(source, target *string, path string) (*apigateway.PatchOperation, bool) {
	if source == nil {
		if target == nil {
			return nil, false
		}
		return &apigateway.PatchOperation{
			Op:    aws.String("add"),
			Path:  aws.String(path),
			Value: aws.String(fmt.Sprintf("%v", *target)),
		}, true
	}
	if target == nil {
		return &apigateway.PatchOperation{
			Op:   aws.String("remove"),
			Path: aws.String(path),
			From: aws.String(fmt.Sprintf("%v", *source)),
		}, true
	}
	if *target == *source {
		return nil, false
	}
	return &apigateway.PatchOperation{
		Op:    aws.String("replace"),
		Path:  aws.String(path),
		From:  aws.String(fmt.Sprintf("%v", *source)),
		Value: aws.String(fmt.Sprintf("%v", *target)),
	}, true
}


func buildFunctionState(context *endly.Context, function *lambda.FunctionConfiguration, restAPI *apigateway.RestApi) data.Map {
	functionState := data.NewMap()
	functionState.Put("arn", function.FunctionArn)
	if ARN, err := arn.Parse(*function.FunctionArn); err == nil {
		functionState.Put("region", ARN.Region)
		functionState.Put("accountID", ARN.AccountID)
	}
	state := context.State()
	state.SetValue("restAPI.ID", *restAPI.Id)
	functionState.Put("name", function.FunctionName)
	state.Put("function", functionState)
	return state
}



