package apigateway

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
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

func pathInt64(source, target *int64, path string) (*apigateway.PatchOperation, bool) {
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

func SetAPIInfo(restAPI *apigateway.RestApi, aMap data.Map) {
	aMap.SetValue("restAPI.SessionID", *restAPI.Id)
}
