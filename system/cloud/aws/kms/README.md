## Key Management Service Service

This service is github.com/aws/aws-sdk-go/service/kms.KMS proxy 

To check all supported method run
```bash
    endly -s="aws/kms"
```


To check method contract run endly -s="aws/sqs" -a=methodName
```bash
    endly -s="aws/kms"
```


#### Usage:

#### Sensitive data encryption

[@secure.yaml](secure.yaml)
```yaml
init:
  password: changeMe
pipeline:
  setupKey:
    credentials: aws
    action: aws/kms:setupKey
    aliasName: alias/myappKey

  encrypt:
    action: aws/ssm:setParameter
    name: myAppPassword
    '@description': my
    overwrite: true
    type: SecureString
    keyId: alias/myappKey
    value: $password
```

