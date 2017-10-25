# Declarative end to end functional testing (endly)

[![Declarative funtional testing for Go.](https://goreportcard.com/badge/github.com/viant/endly)](https://goreportcard.com/report/github.com/viant/endly)
[![GoDoc](https://godoc.org/github.com/viant/endly?status.svg)](https://godoc.org/github.com/viant/endly)

This library is compatible with Go 1.8+

Please refer to [`CHANGELOG.md`](CHANGELOG.md) if you encounter breaking changes.

- [Motivation](#Motivation)
- [Installation](#Installation)
- [Introduction](#Introduction)
- [System services](#Systemservices)
- [Build and deployment services](#Buildservices)
- [Testing services](#Testingservices)
- [Workfow Service](#Workfowservice)

- [Usage](#Usage)
- [Examples](#Examples)
- [License](#License)
- [Credits and Acknowledgements](#Credits-and-Acknowledgements)



## Motivation

This library was developed to enable simple automated declarative end to end functional testing.
Usually an application takes and input to produce some output. Most software would
uses a system to run on it with some services like datastore (rdbms, key-value stores) for application meta data, 
or to process in the required by a business way. Other services used by an application can include caching layer or external API services.
The typical application output could be data in datastore persisted by UI, logs produced by application, performance counters,
or profile data based on used activity.
Finally application needs to be build and deployed with the required service into a system in automated fashion to be tested.
This framework provide end to end capability to eun test from preparing system with its service, building and deploying application to verification 
that expected output has been produced.


<a name="Installation></a>
## Installation

TODO add me

<a name="Introduction></a>
## Introduction

This testing framework uses [Neatly](https://github.com/viant/neatly) format to represent a workflow.


**[Workflow](workflow.go)** an abstraction to define a set of task with its action.

**Task** an abstraction to logically group one or more action, for example, init,test.

**Action** an abstraction defining a call to a neatly service action. 
An action does actual job, like starting service, building and deploying app etc, 

**ActionRequest** an abstraction representing a service request.
        
**ActionResponse** an abstraction representing a service response.

**[Service](service.go)** an abstraction providing set of functionalities triggered by specified action/request.

**State** key/value pair map that is used to mange state during the workflow run. 
The state can be change by providing variable definition.
The workflow content, data structures, can use dollar '$' sign followed by variable name 
to get its expanded to its corresponding state value if the key has been present.

**[Variables](variable.go)** an abstraction having capabilities to change a state map.

Variable has the following attributes
* **Name**: name can be defined as key to be stored in state map or expression with the key
The following expression are supported:

* **Value**: any type value that is used when from value is empty
* From  name of a key state key, or expression with key.    
The following expression are supported:
    * number increments  **++**, for example  counter++, where counter is a key in the state
    * array element shift  **<-**, for example  <-collection, where collection is a key in the state      
    * reference **$** for example $ref, where ref is the key in the state, in this case the value will be 
    evaluated as value stored in key pointed by content of ref variable
    


<a name="Systemservices></a>


## System services


All services are running on the system referred as target and defined as [Resource](https://raw.githubusercontent.com/viant/toolbox/master/url/resource.go)


**Execution services**

The execution service is responsible for opening, managing terminal session, with ability to send command and extract data.

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| exec | open | open SSH session on the target resource. | [OpenSessionRequest](service_exec_session.go) | [OpenSessionResponse](service_exec_session.go) |
| exec | close | closes SSH session | [CloseSessionRequest](service_exec_session.go) | [CloseSessionResponse](service_exec_session.go) |
| exec | command | executes basic commands | [CommandRequest](service_exec_command.go) | [CommandResponse](service_exec_command_response.go) |
| exec | managedCommand | executes commands with ability to extract data, define error or success state | [ManagedCommandRequest](service_exec_command.go) | [SystemCommandResponse](service_system_exec_command_response.go) |



**Daemon service.**

Daemon System service is responsible for managing system daemon services.

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- | 
| daemon | status | check status of system daemon | [DaemonStatusRequest](service_daemon_status.go) | [DaemonInfo](service_daemon_status.go) | 
| daemon | start | start requested system daemon | [DaemonStartRequest](service_daemon_start.go) | [DaemonInfo](service_daemon_status.go) | 
| daemon | stop | stops requested system daemon | [DaemonStopRequest](service_daemon_stop.go) | [DaemonInfo](service_daemon_status.go) | 


**Process service**

Process service is responsible for starting, stopping and checking status of custom application.

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- | 
| process | status | check status of an application | [ProcessStatusRequest](service_process_status.go) | [ProcessStatusResponse](service_process_status.go) | 
| process | start | start provided application | [ProcessStartRequest](service_process_start.go) | [ProcessStartResponse](service_process_start.go) | 
| process | stop | kill requested application | [ProcessStopRequest](service_process_stop.go) | [CommandResponse](exec_command_response.go) | 


**Sdk Service**

Sdk service sets active terminal session with requested sdk version.

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- | 
| sdk | set | sets system with requested sdk and version | [SdkSetRequest](service_sdk_set.go) | [SdkSetResponse](service_sdk_set.go) | 


**Docker service**

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- | 
| docker | run | run requested docker service | [DockerRunRequest](service_docker_run.go) | [DockerContainerInfo](service_docker_container.go) | 
| docker | images | check docker image| [DockerImagesRequest](service_docker_run.go) | [DockerImagesResponse](service_docker_image.go) | 
| docker | pull | pull requested docker image| [DockerPullRequest](service_docker_run.go) | [DockerImageInfo](service_docker_image.go) | 
| docker | process | check docker container processes | [DockerContainerCheckRequest](service_docker_container.go) | [DockerContainerCheckResponse](service_docker_container.go) | 
| docker | container-start | start specified docker container | [DockerContainerStartRequest](service_docker_container.go) | [DockerContainerInfo](service_docker_container.go) | 
| docker | container-command | run command within specified docker container | [DockerContainerCommandRequest](service_docker_container.go) | [CommandResponse](exec_command_response.go) | 
| docker | container-stop | stop specified docker container | [DockerContainerStopRequest](service_docker_container.go) | [DockerContainerInfo](service_docker_container.go) | 
| docker | container-remove | remove specified docker contaienr | [DockerContainerRemoveRequest](service_docker_container.go) | [CommandResponse](exec_command_response.go) | 

TODO add stop (with names of running images to stop in one go)


<a name="Buildservices"></a>
## Build and deployment services



**Transfer service**

Transfer service is responsible for transferring data from the source to the target destination, optionally it supports transferred content data substitution. 

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| transfer | copy | copy one or more resources from the source to target destination | [TransferCopyRequest](service_transfer_copy.go) | [TransferCopyResponse](service_transfer_copy.go) |



**Version Control Service**

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| version/control | status | Runs version control check on provided URL | [VcStatusRequest](service_vc_status.go) | [VcInfo](service_vc_info.go)  |
| version/control | checkout | If target directory already  exist with matching origin URL, this action only pulls the latest changes without overriding local ones, otherwise full checkout | [VcCheckoutRequest](service_vc_checkout.go) | [VcInfo](service_vc_info.go)   |
| version/control | commit | commit commits local changes to the version control | [VcCommitRequest](service_vc_commit.go) | [VcInfo](service_vc_info.go)   |
| version/control | pull | retrieves the latest changes from the origin | [VcPullRequest](service_vc_pull.go) | [VcInfo](service_vc_info.go)   |


**Build service**

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| build | load | Loads BuildMeta for the supplied resource | [BuildLoadMetaRequest](service_build_load.go) | [BuildLoadMetaResponse](service_build_load.go)  |
| build | register | Register BuildMeta in service repo | [BuildRegisterMetaRequest](service_build_register.go) | [BuildRegisterMetaResponse](service_build_register.go)  |
| build | build | Run build for provided specification | [BuildRequest](service_build_build.go) | [BuildResponse](service_build_build.go)  |


**Deployment service** 
Deployment service check if target path resource, the app has been installed with requested version, if not it will transfer it and run all defined commands/transfers.
Maven, tomcat use this service.

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| deployment | deploy | run deployment | [DeploymentDeployRequest](service_deployment_deploy.go) | [DeploymentDeployResponse](service_deployment_deploy.go) |


<a name="Testingservices"></a>
### Testing services

**Http Runner** 

Http runner sends one or more http request to the specified endpoint, it manages cookie with one grouping send request.


| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| http/runner | send | Sends one or more http request to the specified endpoint. | [SendHttpRequest](service_http_runner_send.go) | [SendHttpResponse](service_http_runner_send.go) |


**Rest Runner**

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| rest/runner | send | Sends one rest request to the endpoint. | [RestSendRequest](service_rest_send.go) | [RestSendResponse](service_rest_send.go) |



TODO add Selenium/WebDriver runner


**Generic validation service**

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| validator | assert | performs validation on provided actual  vs expected data structure. | [ValidatorAssertRequest](service_validator_assert.go) | [AssertionInfo](assertion_info.go) |


**Log validation service** 

In order to get log validation, 
   1) register log listener, to dynamically detect any log changes (log shrinking/rollovers is supported), as long as new logs is detected it is ready to be validated.
   2) run log validation. Log validation removes validated logs from the pending queue.
   3) optionally reset listener to discard pending validation logs.


| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| validator/log | listen | starts listening for log file changes on specified location  |  [LogValidatorListenRequest](service_log_validator_listen.go) | [LogValidatorListenResponse](service_log_validator_listen.go)  |
| validator/log | reset | discards logs detected by listener | [LogValidatorResetRequest](service_log_validator_reset.go) | [LogValidatorResetResponse](service_log_validator_reset.go)  |
| validator/log | assert | performs validation on provided expected log records against actual log file records. | [LogValidatorAssertRequest](service_log_validator_assert.go) | [LogValidatorAssertResponse](service_log_validator_assert.go)  |


**Datastore services**

The first action that needs to be run is to register database name with dsc connection config, and optionally init scripts.


| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| dsunit | register | register database connection, and optionally executes init scripts |  [DsUnitRegisterRequest](service_dsunit_register.go) | [DsUnitRegisterResponse](service_dsunit_register.go)  |
| dsunit | mapping |  register virtual mapping between a virtual table and dozen actual tables to simplify setup. |  [DsUnitMappingRequest](service_dsunit_mapping.go) | [DsUnitMappingResponse](service_dsunit_mapping.go)  |
| dsunit | register | register database connection, and optionally executes init scripts |  [DsUnitRegisterRequest](service_dsunit_register.go) | [DsUnitRegisterResponse](service_dsunit_register.go)  |
| dsunit | sequence | takes current sequences for specified tables |  [DsUnitTableSequenceRequest](service_dsunit_sequence.go) | [DsUnitTableSequenceResponse](service_dsunit_sequence.go)  |
| dsunit | prepare | populates database with setup data |  [DsUnitTablePrepareRequest](service_dsunit_prepare.go) | [DsUnitTablePrepareResponse](service_dsunit_prepare.go)  |
| dsunit | expect | verifies database content with expected data |  [DsUnitTableExpectRequest](service_dsunit_prepare.go) | [DsUnitTableExpectResponse](service_dsunit_prepare.go)  |


To simplify setup/verification data process [DsUnitTableData](service_dsunit_data.go) has been introduce, so that data can be push into state, and then transform to the dsunit expected data with AsTableRecords udf function.


<a name="Workfowservice"></a>
## Workfow service


**Workflow Service**

Workflow service provide capability to run task, action from any defined workflow.

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| workflow | load | Loads workflow from provided path | [WorkflowLoadRequest](service_workflow_load.go) | [WorkflowLoadRequest](service_workflow_load.go)  |
| workflow | register | Register provide workflow in registry | [WorkflowLoadRequest](service_workflow_register.go) |  |
| workflow | run | run workflow with specified tasks and parameters | [WorkflowRunRequest](service_workflow_run.go) | [WorkflowRunResponse]((service_workflow_run.go) |



**Workflow Lifecycle**


**Predefined workflows**



 
 #Good practises:

    Test datastructure:
       

         
         	
<a name="License"></a>
## License

The source code is made available under the terms of the Apache License, Version 2, as stated in the file `LICENSE`.

Individual files may be made available under their own specific license,
all compatible with Apache License, Version 2. Please see individual files for details.


<a name="Credits-and-Acknowledgements"></a>

##  Credits and Acknowledgements

**Library Author:** Adrian Witas

