**Endly Google Compute Engine Service**

Provides ability to call operations on  [*compute.Service client](https://cloud.google.com/compute/docs/reference/latest/)

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| gce | call | run gce operation | [CallRequest](gce/service_contract.go) | [CallResponse](gce/service_contract.go)  |

'call' action's service, method and paramters are proxied to [GCE client](https://cloud.google.com/compute/docs/reference/latest/)

