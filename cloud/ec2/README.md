**Amazon Elastic Compute Cloud Service**


Provides ability to call operations on  [EC2 client](https://github.com/aws/aws-sdk-go/tree/master/service/ec2)


## Usage

```go

var request = &ec2.CallRequest{
              		Credentials: "endly-aws-east",
              		Method:      "DescribeInstances",
              		Input: map[string]interface{}{
              			"InstanceIds": []interface{}{
              				"i-0139209d5358e60**",
              			},
              		},
              	}
var response = &ec2.CallResponse{}    
var manager = endly.New()
var context = manager.NewContext(nil)
var err := endly.Run(context, request, response)
if err != nil {
	log.Fatal(err)
}
fmt.Sprintf("%v\n", response);


```

## Endly workflow service action

Run the following command for aws/ec2 service operation details:

```bash
endly -s='aws/ec2' -a=call
```

<a name="endly"></a>

## Endly inline workflow


```bash
endly -r=run
```

@run.yaml

```yaml
pipeline:
  ec2-up:
    action: aws/ec2:run
    credentials: endly-aws-east
    method: StartInstances
    input:
      instanceIds:
        - i-0139209d5358e60**
    
```


| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| aws/ec2 | call | run ec2 operation | [CallRequest](service_contract.go) | [CallResponse](service_contract.go)  |

'call' action's method and input are proxied to [EC2 client](https://github.com/aws/aws-sdk-go/tree/master/service/ec2)


#### Shared/global workflow

- [cloud/ec2](../../shared/workflow/cloud/ec2)

- List workflow tasks:

```bash
 endly -w='cloud/ec2'  -t='?'
```

- Run workflow

```bash
endly -r=start
```

@start.yaml

```yaml
pipeline:
  ec2-up:
    workflow: cloud/ec2
    tasks: start
    ec2InstanceId: i-0139209d5358e60a4
    awsCredential: endly-aws-west
```

@stop.yaml

```yaml
pipeline:
  ec2-down:
    workflow: cloud/ec2
    tasks: stop
    ec2InstanceId: i-0139209d5358e60a4
    awsCredential: endly-aws-west
```

@restart.yaml

```yaml

defaults:
  ec2InstanceId: i-0139209d5358e60a4
  awsCredential: endly-aws-west
pipeline:
  ec2-restart:
    down:
      workflow: cloud/ec2
      tasks: stop
    up:
      workflow: cloud/ec2
      tasks: start
      
```


