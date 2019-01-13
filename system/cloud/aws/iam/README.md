# AWS Identity and Access Management (IAM) Service

This service is github.com/aws/aws-sdk-go/service/iam.IAM proxy 


To check all supported method run
```bash
    endly -s="aws/iam"
```

To check method contract run endly -s="aws/lambda" -a=methodName
```bash
    endly -s="aws/iam" -a=listRoles
```

On top of that service implements the following helper methods:

- recreateRole: drop if exists and create new role
- setupRole: creates or modifies role with supplied policies

### Usage:


#### Setting up role

```bash
endly -r=setup_role
```


@setup_role.yaml
```yaml
pipeline:
 action: aws/iam:setupRole
    credentials: aws
    rolename: myRole
    define:
      - policyname: myPolicy
        policydocument: $Cat('privilege-policy.json')
    attach:
      - policyarn: arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole
```

#### Getting role info


```bash
endly -r=get_role_info
```

@get_role_info.yaml
```yaml
pipeline:
 info:
    action: aws/iam:getRoleInfo
    credentials: aws
    roleName: myRole
```



#### Getting user info


```bash
endly -r=get_user_info
```

@get_user_info.yaml
```yaml
pipeline:
 info:
    action: aws/iam:getUserInfo
    credentials: aws
    roleName: myUser
```


