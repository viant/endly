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

Set subscription

```bash
endly subscription.yaml authWith=myAWSSecret.json
```

[@subscription.yaml](usage/subscription.yaml)
```yaml
pipeline:
  setupLambdaSubscription:
    action: aws/sns:setupSubscription
    protocol: lambda
    endpoint: $functionName
    topic: $topic
```


**Set permission**

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


See also [Message resource setup and testing](https://github.com/viant/endly/tree/master/testing/msg)
