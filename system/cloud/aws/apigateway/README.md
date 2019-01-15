# APIGateway Service

This service is github.com/aws/aws-sdk-go/service/apigateway.APIGateway proxy 

To check all supported method run
```bash
    endly -s="aws/apigateway"
```

To check method contract run endly -s="aws/apigateway" -a=methodName
```bash
    endly -s="aws/apigateway" -a='getRestApis'
```

On top of that service implements the following helper methods:
- setupResetAPI

#### Usage:

#### Setting Rest API

```bash
endly -r=setup
```

@setup.yaml
```yaml
pipeline:
  createAPI:
    action: aws/apigateway:setupRestAPI
    credentials: aws
    '@name': ipLookupAPI
    resources:
      - path: /ip
        methods:
          - httpMethod: ANY
            functionName: lookupIp

  info:
    action: print
    message: $AsJSON(${createAPI})
```

