**Deployment service** 
Deployment service checks if target path resource, the app has been installed with requested version, if not it will transfer it and run all defined commands/transfers.
Maven, tomcat use this service.

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| deployment | deploy | run deployment | [DeployRequest](service_contract.go) | [DeployResponse](service_contract.go) |

