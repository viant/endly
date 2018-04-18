**Http runner service**

Http runner sends one or more HTTP request to the specified endpoint; 
it manages cookie within [SendRequest](service_contract.go).

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| http/runner | send | Sends one or more http request to the specified endpoint. | [SendRequest](service_contract.go) | [SendResponse](service_contract.go) |
