# Lambda Service

This service is github.com/aws/aws-sdk-go/service/lambda.Lambda proxy 


To check all supported method run
```bash
    endly -s="aws/lambda"
```

To check method contract run endly -s="aws/lambda" -a=methodName
```bash
    endly -s="aws/lambda" -a=createFunction
```

On top of that service implements the following helper methods:

- recreateFunction: drop if exists and create new function
- dropFunction: drop function with dependencies
- setupPermission: add permission if it does not exists
- setupFunction: creates or modifies function with specified policies

### Usage:

#### Create function


```bash
endly -r=setup
```


@setup.yaml
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






#### Invoke function


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