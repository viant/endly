package s3

import (
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/viant/endly/service/system/cloud/aws"
)

func indexLambdaFunction(configurations []*s3.LambdaFunctionConfiguration) map[string]*s3.LambdaFunctionConfiguration {
	var result = make(map[string]*s3.LambdaFunctionConfiguration)
	for _, config := range configurations {
		key, _ := aws.ArnName(*config.LambdaFunctionArn)
		result[key] = config
	}
	return result
}
