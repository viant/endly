# Simple Notidication Service Service

This service is github.com/aws/aws-sdk-go/service/sqs.SQS proxy 

To check all supported method run
```bash
    endly -s="aws/sns"
```

To check method contract run endly -s="aws/sqs" -a=methodName
```bash
    endly -s=aws/sns:listSubscriptions
    endly -s=aws/sns:setupPermission
```

#### Usage:

```bash
endly set_permission.yaml authWith=myAWSSecret.json
``` 


[@set_permission.yaml](usage/set_permission.yaml)
```yaml
init:
  '!awsCredentials': $params.authWith

pipeline:
  setupPermission:
    action: aws/sns:setupPermission
    credentials: $awsCredentials
    queue: ms-dataflowStorageMirrorQueue
    AWSAccountId:
      - ${aws.accountID}
    actionName:
      - 'publish'
    everybody: true 
```