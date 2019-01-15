package lambda

import (
	"crypto/sha256"
	"encoding/base64"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/viant/endly"
	"github.com/viant/toolbox/data"
)

func hasDataChanged(data []byte, dataSha256 string) bool {
	algorithm := sha256.New()
	algorithm.Write(data)
	codeSha1 := algorithm.Sum(nil)
	dataSha1BAse64Encoded := base64.URLEncoding.EncodeToString(codeSha1)
	return dataSha256 != dataSha1BAse64Encoded
}

func GetFunctionConfiguration(context *endly.Context, functionName string) (*lambda.FunctionConfiguration,  error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	functionOutput, err := client.GetFunction(&lambda.GetFunctionInput{
		FunctionName: &functionName,
	})
	if err != nil {
		return nil, err
	}
	return functionOutput.Configuration, nil
}

func SetFunctionInfo(function *lambda.FunctionConfiguration, aMap data.Map) {
	functionState := data.NewMap()
	functionState.Put("arn", function.FunctionArn)
	if ARN, err := arn.Parse(*function.FunctionArn); err == nil {
		functionState.Put("region", ARN.Region)
		functionState.Put("accountID", ARN.AccountID)
	}
	functionState.Put("name", function.FunctionName)
	aMap.Put("function", functionState)
}