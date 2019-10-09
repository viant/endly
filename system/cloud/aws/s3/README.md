# S3 Service

This service is github.com/aws/aws-sdk-go/service/s3.S3 proxy 

To check all supported method run
```bash
    endly -s="aws/s3"
```

To check method contract run endly -s="aws/s3" -a=methodName
```bash
    endly -s="aws/s3" -a='listObjects'
```



On top of that service implements setupBucketNotification to configure bucket lambda, topic or queue notification. 



## Usage:

### Lambda s3 notification

```bash
endly deploy.yaml authWith=aws-secrets.json
```

_where:_ 
- [@deploy.yaml](usage/lambda/deploy.yaml)

```yaml
  setLambdaNotification:
    action: aws/s3:setupBucketNotification
    credentials: $awsCredentials
    bucket: ${myBucket}
    lambdaFunctionConfigurations:
      - functionName: $functionName
        id: ObjectCreatedEvents-${functionName}-${myBucket}
        events:
          - s3:ObjectCreated:*
        filter:
          prefix:
            - folder1
          suffix:
            - .csv

```
- [@privilege-policy.json](usage/lambda/privilege-policy.json)
- [main.go](usage/lambda/main.go)
- [@aws-secrets.json](usage/lambda/aws-secrets.json) 


### Queue s3 notification


```bash
endly deploy.yaml authWith=aws-secrets.json
```

_where:_ 
- [@deploy.yaml](usage/sqs/deploy.yaml)

```yaml
  setBucketQueueNotification:
    action: aws/s3:setupBucketNotification
    bucket: ${bucket}
    queueConfigurations:
      - queue: $queue
        id: ObjectCreatedEvents
        events:
          - s3:ObjectCreated:*
```

- [@privilege-policy.json](usage/sqs/privilege-policy.json)
- [main.go](usage/sqs/main.go)
- [@aws-secrets.json](usage/sqs/aws-secrets.json)



### Topic s3 notification


```bash
endly deploy.yaml authWith=aws-secrets.json
```

_where:_ 
- [@deploy.yaml](usage/sqs/deploy.yaml)

```yaml
  setBucketTopicNotification:
    action: aws/s3:setupBucketNotification
    credentials: $awsCredentials
    bucket: ${triggerBucket}
    topicConfigurations:
      - topic: $topic
        id: ObjectCreatedEvents
        events:
          - s3:ObjectCreated:*

```

- [@privilege-policy.json](usage/sqs/privilege-policy.json)
- [main.go](usage/sqs/main.go)
- [@aws-secrets.json](usage/sqs/aws-secrets.json)
