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

On top of that service implements the following helper methods:
 - setupBucketNotification


#### Usage:

##### Lambda s3 notification deployment and setup

```bash
endly -r=deploy
```

@deploy.yaml
```yaml
init:
  functionRole: lambda-filemeta-executor
  functionName: FilemetaFn
  codeZip: ${appPath}filemeta/app/filemeta.zip
  privilegePolicy: ${parent.path}/privilege-policy.json
pipeline:
  deploy:
    build:
      action: exec:run
      target: $target
      sleepTimeMs: 1500
      errors:
        - ERROR
      commands:
        - cd ${appPath}filemeta/app
        - unset GOPATH
        - export GOOS=linux
        - export GOARCH=amd64
        - go build -o filemeta
        - zip -j filemeta.zip filemeta

    setupFunction:
      action: aws/lambda:deploy
      credentials: $awsCredentials
      functionname: $functionName
      runtime:  go1.x
      handler: filemeta
      code:
        zipfile: $LoadBinary(${codeZip})
      rolename: lambda-filemeta-executor
      define:
        - policyname: s3-mye2e-bucket-role
          policydocument: $Cat('${privilegePolicy}')
      attach:
        - policyarn: arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole

    notification:
      action: aws/s3:setupBucketNotification
      credentials: $awsCredentials
      sleepTimeMs: 10000
      bucket: mye2e-bucket2
      lambdaFunctionConfigurations:
        - functionName: $functionName
          id: ObjectCreatedEvents
          events:
            - s3:ObjectCreated:*
          filter:
            prefix:
              - folder1
            suffix:
              - .csv
```