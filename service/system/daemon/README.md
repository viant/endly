**Daemon service.**

Daemon System service is responsible for managing system daemon services.

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- | 
| daemon | status | check status of system daemon | [StatusRequest](service_contract.go) | [Info](service_contract.go) | 
| daemon | start | start requested system daemon | [StartRequest](service_contract.go) | [Info](service_contract.go) | 
| daemon | stop | stop requested system daemon | [StopRequest](service_contract.go) | [Info](service_contract.go) | 
