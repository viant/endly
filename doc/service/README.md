# Service


Service is actual execution unit, in order to run any workflow's action or piple task you have use specify :
service ID, action, and actual action request.

For instance to copy some assets from S3 to some remote box you can use one of the following:

### Inline pipeline task
```bash
endly -r=copy
```
@copy.yaml
```yaml
pipeline:
  transfer:
    action: storage:copy  
    source:
      URL: s3://mybucket/dir
      credentials: aws-west
    dest:
      URL: scp://dest/dir2
      credential: dest
    assets:
      file1.txt:
      file2.txt: renamedFile2      

```

In this case action selector specifies service:action, while the other keys define action actual request data structure.


### Workflow

```bash
endly -w=test
```


@test.csv

|Workflow|Name|Tasks| | |
|---|---|---|---| --- |
| |test|%Tasks|| |
|**[]Tasks**|**Name**|**Actions**| |
| |transfer|%Transfer| |
|**[]Transfer**|**Service**|**Action**|**Request**|**Description**|
| |storage|copy|@transfer| copy asset |



@transfer.yaml
```yaml
source:
  URL: s3://mybucket/dir
  credentials: aws-west
dest:
  URL: scp://dest/dir2
  credential: dest
assets:
  file1.txt:
  file2.txt: renamedFile2      
```




To get the latest list of endly supported services run
```text
endly -s='*'
```

To check all actions supported by given service run 
`endly -s='[service name]'`

i.e 
```text
endly -s='docker'
```

To check request/response for service/action combination run 
`endly -s='[service name]' -a=[action name]`

i.e 
```text
endly -s='docker' -a='run'
```




Endly services implement [Service](service.go) interface.
The following diagram shows service with its component.


![Service diagram](diagram.png)


1) **System services**
    - [SSH Executor Service](../../system/exec)
    - [Storage Service](../../system/storage)
    - [Process Service](../../system/process)
    - [Daemon Service](../..//system/daemon)
    - [Network Service](../../system/network)
    - [Docker Service](../../system/docker)
2) **Cloud services**
    - [Amazon Elastic Compute Cloud Service](../../cloud/ec2)
    - [Google Compute Engine Service](../../cloud/gce)
3) **Build and Deployment Services**
    - [Sdk Service](../../deployment/sdk)
    - [Version Control Service](../../deployment/vc)
    - [Build Service](../../deployment/build)
    - [Deplyment Service](../../deployment/deploy)
6) **Testing Services**
   - [Validator](../../testing/validator)
   - [Log Validator Service](../../testing/log)
   - [Datastore Preparation and Validation Service](../../testing/dsunit)
   - **Endpoint Services**
      - [Http Endpoint Service](../../testing/endpoint/http) 
   - **Runner Services**
      - [Http Runner Service](../../testing/runner/http) 
      - [REST Runner Service](../../testing/runner/rest) 
      - [Selenium Runner Service](../../testing/runner/selenium) 

   
7) **Notification Services**
   - [SMTP Service](../../notify/smtp)
8) **Workflow service**
   - [Workflow Service](../../workflow/)
 



<a name="new_service>&nbsp;</a>
# Adding new service

The following step provide quick instruction how to add new endly service:

- Create a service contract for each service operation for instance 'xx' request/response may look like the following:
```go
type XXRequest struct {
	SomeField string
}

type XXResponse struct {
	SomeOtherField string
}
```
- Create a new service type that embeds *AbstractService
```go
type xxService struct {
	*AbstractService
}
```
- Provide implementation for each action i.e.
```go
func (s *xxService) xx(request *XXRequest) (*XXResponse,error) {
	var response = &XXResponse{}
	var err error
	//some logic here
	
	return response, err
}
````
- Register service routes for each action
```go
func (s *xxService) registerRoutes() {
	
	//xx action route
	s.Register(&Route{
		Action: "xx",
		RequestInfo: &ActionInfo{
			Description: "xx action ....",
		},
		RequestProvider: func() interface{} {
			return &XXRequest{}
		},
		ResponseProvider: func() interface{} {
			return &XXResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*XXRequest); ok {
				return s.xx(context, request)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
	
	
}
```
- Create service constructor
```go
func newXXService() Service {
	var result = &xxService{
		AbstractService: NewAbstractService("xx"),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
```
- Register a new service with endly repository.
```go
import "github.com/viant/endly"

func init() {
	endly.Registry.Register(func() endly.Service {
		return New()
	})
}
```
- Add a new service package to [bootstrap](./../../bootstrap/bootstrap.go) import.
