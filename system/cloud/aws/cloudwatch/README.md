# Amazon Cloud Watch service

This service is github.com/aws/aws-sdk-go/service/ec2.EC2 proxy 

To check all supported method run
```bash
    endly -s="aws/cloudwatch"
```

To check method contract run endly -s="aws/cloudwatch" -a=methodName
```bash
    endly -s="aws/cloudwatch" -a='listMetrics'
```

#### Usage:

```yaml
pipeline:
  task1:
    action: aws/cloudwatch:listMetrics
    credentials: aws
    dimensions:
      - name: LogGroupName
        value: /aws/lambda/MsgLogFn

```