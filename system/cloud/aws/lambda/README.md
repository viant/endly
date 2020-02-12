# Lambda Service

- [Usage](#usage)
  - [Deployment](#deployment)
    - [Basic](#basic)
    - [S3](#s3)
    - [SQS](#sqs)
    - [SNS](#sns)
    - [APIGateway](#api-gateway)
    - [VPC](#vpc)
    - [Scheduled](#scheduled)
  - [Function invocation](#function-invocation)

This service is github.com/aws/aws-sdk-go/service/lambda.Lambda proxy 


To check all supported method run
```bash
    endly -s="aws/lambda"
```

To check method contract run endly -s="aws/lambda" -a=methodName
```bash
    endly -s="aws/lambda" -a=deployFunction
```

On top of that service implements the following helper methods:

- deployFunction: creates or modifies function with specified policies
- recreateFunction: drop if exists and create new function
- dropFunction: drop function with dependencies
- setupPermission: add permission if it does not exists

## Usage

Prerequisites:

[AWS credentials](https://github.com/viant/endly/tree/master/doc/secrets#aws)

#### Deployment

##### Basic

```endly deploy```

[@deploy.yaml](usage/basic/deploy.yaml)
```yaml
init:
  functionRole: lambda-helloworld-executor
  functionName: HelloWorld
  codeZip: /tmp/hello/main.zip
  awsCredentials: aws
  privilegePolicy: privilege-policy.json
pipeline:
  build:
    action: exec:run
    target: $target
    sleepTimeMs: 1500
    checkError: true
    commands:
      - cd ${appPath}helloworld/app
      - unset GOPATH
      - export GOOS=linux
      - export GOARCH=amd64
      - go build -o helloworld
      - zip -j helloworld.zip helloworld

    deploy:
      action: aws/lambda:deploy
      credentials: $awsCredentials
      functionname: $functionName
      runtime:  go1.x
      handler: helloworld
      code:
        zipfile: $LoadBinary(${codeZip})
      rolename: lambda-helloworld-executor
      define:
        - policyname: lambda-helloworld-executor-role
          policydocument: $Cat('${privilegePolicy}')
      attach:
        - policyarn: arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole
```

##### S3

```endly deploy```

[@deploy.yaml](usage/s3/deploy.yaml)
```yaml
init:
  functionRole: lambda-mystoragefunction-executor
  functionName: MyStorageFuncton
  codeZip: ${appPath}mystoragefunction/app/mystoragefunction.zip
  privilegePolicy: privilege-policy.json
  myBucket: testBucket
  
pipeline:
  build:
    action: exec:run
    target: $target
    sleepTimeMs: 1500
    checkError: true
    commands:
      - cd ${appPath}mystoragefunction/app
      - unset GOPATH
      - export GOOS=linux
      - export GOARCH=amd64
      - go build -o mystoragefunction
      - zip -j mystoragefunction.zip mystoragefunction

  deploy:
    action: aws/lambda:deploy
    credentials: $awsCredentials
    functionname: $functionName
    runtime:  go1.x
    handler: mystoragefunction
    code:
      zipfile: $LoadBinary(${codeZip})
    rolename: lambda-mystoragefunction-executor
    define:
      - policyname: s3-${testBucketPrefix}2-role
        policydocument: $Cat('${privilegePolicy}')
    attach:
      - policyarn: arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole

  notification:
    action: aws/s3:setupBucketNotification
    credentials: $awsCredentials
    sleepTimeMs: 20000
    bucket: ${myBucket}
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

##### SQS

```endly deploy```

[@deploy.yaml](usage/sqs/deploy.yaml)
```yaml
init:
  functionRole: lambda-mysqsfunction-executor
  functionName: MySQSFunction
  codeZip: ${appPath}mysqsfunction/mysqsfunction.zip
  privilegePolicy: privilege-policy.json
pipeline:
  deploy:
    build:
      action: exec:run
      target: $target
      sleepTimeMs: 1500
      checkError: true
      commands:
        - cd ${appPath}mysqsfunction
        - unset GOPATH
        - export GOOS=linux
        - export GOARCH=amd64
        - go build -o mysqsfunction
        - zip -j mysqsfunction.zip mysqsfunction

    setupFunction:
      action: aws/lambda:deploy
      credentials: $awsCredentials
      functionname: $functionName
      runtime:  go1.x
      handler: mysqsfunction
      code:
        zipfile: $LoadBinary(${codeZip})
      rolename: lambda-mysqsfunction-executor
      define:
        - policyname: sqs-my-queue-role
          policydocument: $Cat('${privilegePolicy}')
      attach:
        - policyarn: arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole
      triggers:
        - source: myQueue
          type: sqs
          enabled: true
          batchSize: 10
  
```

##### SNS

```endly deploy```

[@deploy.yaml](usage/sns/deploy.yaml)
```yaml
init:
  functionRole: lambda-mysnsfunc-executor
  functionName: AggFn
  codeZip: ${appPath}mysnsfunc/app/mysnsfunc.zip
  privilegePolicy: privilege-policy.json
pipeline:
  deploy:
    build:
      action: exec:run
      target: $target
      sleepTimeMs: 1500
      checkError: true
      commands:
        - cd ${appPath}mysnsfunc/app
        - unset GOPATH
        - export GOOS=linux
        - export GOARCH=amd64
        - go build -o mysnsfunc
        - zip -j mysnsfunc.zip mysnsfunc

    deployFunction:
      action: aws/lambda:deploy
      credentials: $awsCredentials
      functionname: $functionName
      runtime:  go1.x
      handler: mysnsfunc
      code:
        zipfile: $LoadBinary(${codeZip})
      rolename: lambda-mysnsfunc-executor
      define:
        - policyname: lambda-sns-execution-role
          policydocument: $Cat('${privilegePolicy}')
      attach:
        - policyarn: arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole

    setupSubscription:
      action: aws/sns:setupSubscription
      protocol: lambda
      endpoint: $functionName
      topic: myTopic
```



##### Api Gateway

```endly deploy```

[@deploy.yaml](usage/apigateway/deploy.yaml)

```yaml
init:
  functionRole: lambda-myapigwfunc-executor
  functionName: DsTransferFn
  codeZip: ${appPath}myapigwfunc/app/myapigwfunc.zip
  privilegePolicy: privilege-policy.json

pipeline:
  deploy:
    build:
      action: exec:run
      target: $target
      sleepTimeMs: 1500
      checkError: true
      commands:
        - cd ${appPath}myapigwfunc/app
        - unset GOPATH
        - export GOOS=linux
        - export GOARCH=amd64
        - go build -o myapigwfunc
        - zip -j myapigwfunc.zip myapigwfunc

    deployFunction:
      action: aws/lambda:deploy
      credentials: $awsCredentials
      functionname: $functionName
      runtime:  go1.x
      handler: myapigwfunc
      timeout: 360
      environment:
        variables:
          CONFIG: $AsString($config)
      code:
        zipfile: $LoadBinary(${codeZip})
      rolename: lambda-myapigwfunc-executor
      define:
        - policyname: myapigwfunce2e-role
          policydocument: $Cat('${privilegePolicy}')
      attach:
        - policyarn: arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole

    setupAPI:
      action: aws/apigateway:deployAPI
      credentials: $awsCredentials
      '@name': myapigwfuncAPI
      resources:
        - path: /{proxy+}
          methods:
            - httpMethod: ANY
              functionname: $functionName
    sleepTimeMs: 15000
post:
  endpointURL: ${setupAPI.EndpointURL}
```


##### Vpc

[@deploy.yaml](usage/vpc/deploy.yaml)
```yaml
init:
  functionRole: lambda-myvpcfunc-executor
  functionName: MyVpcFunc
  codeZip: ${appPath}/myvpcfunc/app/myvpcfunc.zip
  privilegePolicy: privilege-policy.json
pipeline:

  build:
    action: exec:run
    target: $target
    sleepTimeMs: 1500
    checkError: true
    commands:
      - cd ${appPath}/myvpcfunc/app
      - unset GOPATH
      - export GOOS=linux
      - export GOARCH=amd64
      - go build -o myvpcfunc
      - zip -j myvpcfunc.zip myvpcfunc

  deploy:
    action: aws/lambda:deploy
    credentials: $awsSecrets
    functionname: $functionName
    runtime:  go1.x
    handler: myvpcfunc
    environment:
      variables:
        CONFIG: $AsString($myvpcfuncConfig)
    code:
      zipfile: $LoadBinary(${codeZip})
    rolename: lambda-myvpcfunc-executor
    define:
      - policyname: ${myvpcfuncConfig}-role
        policydocument: $Cat('${privilegePolicy}')
    attach:
      - policyarn: arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole
      - policyarn: arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole
    vpcMatcher:
      instance:
        name: myHostWithVpcTemplateSettings
        # vpcConfig:
        #  securityGroupIds:
        #   - sg-XXXXXXX
        #subnetIds:
        # - subnet-YYYYYY
    timeout: 900
    memorySize: 200

```



##### Scheduled

```endly deploy```

[@deploy.yaml](usage/scheduled/deploy.yaml)
```yaml
init:
  functionRole: lambda-scheduled-executor
  functionName: ScheduledFn
  codeZip: ${appPath}scheduled/scheduled.zip
  privilegePolicy: ${parent.path}/privilege-policy.json
pipeline:
  deploy:
    build:
      action: exec:run
      target: $target
      sleepTimeMs: 1500
      checkError: true
      commands:
        - cd ${appPath}scheduled
        - unset GOPATH
        - export GOOS=linux
        - export GOARCH=amd64
        - go build -o scheduled
        - zip -j scheduled.zip scheduled

    setupFunction:
      action: aws/lambda:deploy
      credentials: $awsCredentials
      functionname: $functionName
      runtime:  go1.x
      handler: scheduled
      code:
        zipfile: $LoadBinary(${codeZip})
      schedule:
        expression: rate(1 minute)
      rolename: lambda-scheduled-executor
      attach:
        - policyarn: arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole

```

##### Recreate function


```endly deploy_brute_force```
[@deploy_brute_force.yaml](usage/recreate/deploy.yaml)
```yaml
init:
  functionRole: lambda-helloworld-executor
  functionName: HelloWorld
  codeZip: /tmp/hello/main.zip
  awsCredentials: aws
pipeline:
  deploy:
    build:
      action: exec:run
      target: $target
      sleepTimeMs: 1500
      errors:
        - ERROR
      commands:
        - cd /tmp/hello
        - export GOOS=linux
        - export GOARCH=amd64
        - go build -o main
        - zip -j main.zip main
    createRole:
      credentials: $awsCredentials
      action: aws/iam:recreateRole
      rolename: $functionRole
      assumerolepolicydocument: $Cat('/tmp/hello/trust-policy.json')
    attachPolicy:
      action: aws/iam:attachRolePolicy
      comments: attaching policy to ${createRole.Role.Arn}
      rolename: $functionRole
      policyarn: arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole
      sleepTimeMs: 10000
    createFunction:
      action: aws/lambda:recreateFunction
      role: $createRole.Role.Arn
      functionname: ${functionName}
      runtime:  go1.x
      handler: main
      code:
        zipfile: $LoadBinary($codeZip)
```


#### Function invocation


```bash
endly -r=trigger
```


@trigger.yaml

```yaml
init:
  functionName: HelloWorld
  awsCredentials: aws
pipeline:
  trigger:
    action: aws/lambda:invoke
    credentials: $awsCredentials
    comments: call $functionName lambda function
    functionname: $functionName
    payload: ""
    post:
      payload: $AsString($Payload)
  assert:
    action: validator:assert
    comments: 'validate function output: $payload '
    actual: $payload
    expected: /Hello World/
```