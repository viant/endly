**Endly Google Compute Engine Service**

Provides ability to call operations on  [*compute.Service client](https://cloud.google.com/compute/docs/reference/latest/)


## Usage

```go

var request = &gce.CallRequest{
             			Credentials: "gce",
             			Service:     "Instances",
             			Method:      "Get",
             			Parameters:  []interface{}{"myproject", "us-west1-b", "instance-1"},
             		}
var response = &gce.CallResponse{}    
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
endly -s='gce' -a=call

```

<a name="endly"></a>

## Endly inline workflow


```bash
endly -r=start
```


@start.yaml
```yaml
pipeline:
  gce-up:
    credentials: gce
    action: gce:call
    service: Instances
    method: Get
    parameters:
      - myproject
      - us-west1-b
      - instance-1
```   



| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| gce | call | run gce operation | [CallRequest](gce/service_contract.go) | [CallResponse](gce/service_contract.go)  |

'call' action's service, method and paramters are proxied to [GCE client](https://cloud.google.com/compute/docs/reference/latest/)




#### Shared/global workflow

- [cloud/gce](../../shared/workflow/cloud/gce)

- List workflow tasks:

```bash
 endly -w='cloud/gce'  -t='?'
```


- Run workflow

```bash
endly -r=start
```

@start.yaml

```yaml
pipeline:
  gce-up:
    workflow: cloud/gce
    tasks: start
    gceCredential: gce
    gceProject: abstractmeta-p1
    gceZone: us-west1-b
    gceInstance: instance-1
```


