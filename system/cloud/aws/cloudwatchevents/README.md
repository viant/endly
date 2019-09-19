# Amazon Cloud Watch service

This service is github.com/aws/aws-sdk-go/service/ec2.EC2 proxy 

To check all supported method run
```bash
    endly -s="aws/cloudwatchevents"
```

To check method contract run endly -s="aws/cloudwatchevents" -a=methodName
```bash
    endly -s="aws/cloudwatchevents" -a='listRules'
```

#### Usage:

**List Rules**

```endly aws/cloudwatchevents:listRules credentials=aws-e2e```

or with endly workflow

```endly list_rules.yaml```

[@list_rules.yaml](usage/list_rules.yaml)
```yaml
pipeline:
  list:
    action: aws/cloudwatchevents:listRules
    credentials: aws
```

**Get Rule Info**

```endly aws/cloudwatchevents:getRule credentials=aws name=MyRule```

or with endly workflow

``` endly get_rule.yaml```

[@get_rule.yaml](usage/get_rule.yaml)

```yaml
pipeline:
  list:
    action: aws/cloudwatchevents:getRule
    name: MyRule
    credentials: aws
```

**Deploy scheduled rule**

``` endly deploy_rule```

[@deploy_rule.yaml](usage/deploy_rule.yaml)
```yaml
init:
  event:
    vendor: myvendor
    event: myevent

pipeline:
  deploy:
    action: aws/cloudwatchevents:deployRule
    credentials: aws
    '@name': MyRule
    scheduleexpression: rate(1 minute)
    roleName: AggFnSchduler
    targets:
      - function: AggFn


  putEvent:
    action: aws/cloudwatchevents:putEvents
    entries:
      - source: com.company.app
        detailType: appRequestSubmitted
        detail: $AsJSON($event)
        resource:
          - $deploy.Rule.Arn
```