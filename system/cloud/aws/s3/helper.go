package s3

import (
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/s3"
)

func indexLambdaFunction(configurations []*s3.LambdaFunctionConfiguration) map[string]*s3.LambdaFunctionConfiguration {
	var result = make(map[string]*s3.LambdaFunctionConfiguration)
	for _, config := range configurations {
		ARN, _ := arn.Parse(*config.LambdaFunctionArn)
		result[ARN.Resource] = config
	}
	return result
}
