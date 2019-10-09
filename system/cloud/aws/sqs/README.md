# Simple Queue Service Service

This service is github.com/aws/aws-sdk-go/service/sqs.SQS proxy 

To check all supported method run
```bash
    endly -s="aws/sqs"
```

To check method contract run endly -s="aws/sqs:methodName"

```bash
    endly -s=aws/sqs:listQueues
    endly -s=aws/sqs:setupPermission
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
    action: aws/sqs:setupPermission
    credentials: $awsCredentials
    queue: ms-dataflowStorageMirrorQueue
    AWSAccountIds:
      - ${aws.accountID}
    actions:
      - '*'
    everybody: true
```