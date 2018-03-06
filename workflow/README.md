
<a name="Workfowservice"></a>
## Workflow service


**Workflow Service**

Workflow service provide capability to run task, action from any defined workflow.

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| workflow | run | run workflow with specified tasks and parameters | [RunRequest](service_workflow_contract.go) | [RunResponse](service_workflow_contract.go) |
| workflow | goto | switch current execution to the specified task on current workflow | [GotoRequest](service_workflow_goto.go) | [GotoResponse](service_workflow_contract.go) 
| workflow | switch | run matched  case action or task  | [SwitchRequest](service_workflow_contract.go) | [SwitchResponse](service_workflow_contract.go) |
| workflow | exit | terminate execution of active workflow (caller) | n/a | n/a |
| workflow | fail | fail  workflow | [FailRequest](service_workflow_contract.go) | n/a  |


**Predefined workflows**


<a name="predefined_workflows">	</a>
**Predefined workflows**



[Workflows](shared/workflow)

| Name | Task |Description | 
| --- | --- | --- |
| dockerized_mysql| start | start mysql docker container  |
| dockerized_mysql| stop | stop mysql docker container 
| dockerized_aerospike| start | aerospike mysql docker container |
| dockerized_aerospike| stop | stop aerospike docker container |
| dockerized_memcached| start | aerospike memcached docker container |
| dockerized_memcached| stop | stop memcached docker container |
| tomcat| install | install tomcat |
| tomcat| start | start tomcat instance|
| tomcat| stop | stop tomcat instance |
| vc_maven_build | checkout | checkout the latest code from version control |
| vc_maven_build | build | build the checked out code |
| vc_maven_module_build | checkout | check out all required projects to build a module |
| vc_maven_module_build | build | build module |
| ec2 | start | start ec2 instance |
| ec2 | stop | stop  ec2 instance |
| gce | start | start gce instance |
| gce | stop | stop  gce instance |
| notify_error | notify | send error |
 
 
 
 <a name="predefined_requests"></a>
 **Predefined workflow run requests**
 
 [Requests](shared/requests)
  
 | Name | Workflow | 
 | --- | --- | 
 | [tomcat.json](/shared/req/tomcat.json) | tomcat | 
 | [aerospike.json](/sharedreq/aerospike.json)| dockerized_aerospike |
 | [mysql.json](/sharedreq/mysql.json)| dockerized_mysql |
 | [memcached.json](/sharedreq/memcached.json)| dockerized_memcached|
 | [ec2.json](/sharedreq/ec2.json)| ec2 |
 | [gce.json](/sharedreq/gce.json)| gce |
 | [notify_error.json](/sharedreq/notify_error.json)| notify_error |
 
