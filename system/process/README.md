

**Process service**

Process service is responsible for starting, stopping and checking the status of a custom application.

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- | 
| process | status | check status of an application | [StatusRequest](service_contract.go) | [StatusResponse](service_contract.go) | 
| process | start | start provided application | [StartRequest](service_contract.go) | [StartResponse](service_contract.go) | 
| process | stop | kill requested application | [StopRequest](service_contract.go) | [RunResponse](../exec/service_contract.go) | 

