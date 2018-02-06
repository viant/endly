# Declarative end to end functional testing (endly)

[![Declarative funtional testing for Go.](https://goreportcard.com/badge/github.com/viant/endly)](https://goreportcard.com/report/github.com/viant/endly)
[![GoDoc](https://godoc.org/github.com/viant/endly?status.svg)](https://godoc.org/github.com/viant/endly)

This library is compatible with Go 1.8+

Please refer to [`CHANGELOG.md`](CHANGELOG.md) if you encounter breaking changes.

- [Motivation](#Motivation)
- [Installation](#Installation)
- [Introduction](#Introduction)
- [System services](#SystemServices)
- [Build and deployment services](#Buildservices)
- [Testing services](#Testingservices)
- [Workfow Service](#Workfowservice)
- [Usage](#Usage)
- [Best Practice](#BestPractice)
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
This framework provide end to end capability to test from system preparation with its service, building and deploying application to verification 
that expected output has been produced.


<a name="Installation"></a>
## Installation

```text
#optionally set GOPATH directory
export GOPATH=/Projects/go

go get -u github.com/viant/endly


```


<a name="Introduction"></a>
## Introduction


Typical web application automated functional can be broken down as follow:

1) System preparation 
    1) System services initialization.  (RDBM, NoSQL, Caching 3rd Party API)
    2) Application container initialization if application uses it (Application server)
2) Application build and deployment
    1) Application code checkout.
    2) Application build
    3) Application deployment
3) Testing    
    1) Preparing test data
    2) Actual application testing
        1) Http runner
        2) Reset runner
        3) Selenium runner
    3) Application output verification
    4) Application modified data verification
    5) Application produced log verification    
4) Cleanup
    1) Data cleanup 
    2) Application shutdown
    3) Application system services shutdown 
    



This testing framework uses [Neatly](https://github.com/viant/neatly) format to represent a workflow.


**[Workflow](workflow.go)** an abstraction to define a set of task with its action.

**Task** an abstraction to logically group one or more action, for example, init,test.

**Action** an abstraction defining a call to a service. 
An action does actual job, like starting service, building and deploying app etc, 

**ActionRequest** an abstraction representing a service request.
        
**ActionResponse** an abstraction representing a service response.

**[Service](service.go)** an abstraction providing set of functionalities triggered by specified action/request.

**State** key/value pair map that is used to mange state during the workflow run. 
The state can be change by providing variable definition.
The workflow content, data structures, can use dollar '$' sign followed by variable name 
to get its expanded to its corresponding state value if the key has been present.

**[Variables](variable.go)** an abstraction having capabilities to change a state map.

A workflow variable defines data transition between input and output state map.


Variable has the following attributes
* **Name**: name can be defined as key to be stored in state map or expression 
     * array element push **->**, for instance ->collection, where collection is a key in the state map      
    * reference **$** for example $ref, where ref is the key in the state, in this case the value will be 

* **Value**: any type value that is used when from value is empty
* From  name of a key state key, or expression with key.    
The following expression are supported:
    * number increments  **++**, for example  counter++, where counter is a key in the state
    * array element shift  **<-**, for example  <-collection, where collection is a key in the state      
    * reference **$** for example $ref, where ref is the key in the state, in this case the value will be 
    evaluated as value stored in key pointed by content of ref variable
    

**Variable in actions:**


| Operation | Variable.Name | Variable.Value | Variable.From | Input State Before | Input State After | Out State Before | Out State  After |
| --- | --- | --- | ---- | --- | --- | --- | --- |
| Assignment | key1 | [1,2,3] | n/a | n/a | n/a | { } |{"key1":[1,2,3]}|
| Assignment by reference | $key1  | 1 | n/a| {"key1":"a"} | n/a | { } | {"a":1} |
| Assignment | key1 | n/a | params.k1 | {"params":{"k1":100}} | n/a | { } | {"key1":100} |
| Assignment by reference | key1  | n/a | $k | {"k":"a", "a":100} |n/a |  { } | {"key1":100} |
| Push | ->key1 | 1 | n/a | n/a | n/a | { } | {"key1":[1]} | 
| Push | ->key1 | 2 | n/a | n/a | n/a | {"key1":[1]} | {"key1":[1,2]} | 
| Shift | item | n/a  | <-key1 | n/a | n/a | {"key1":[1, 2]} | {"key1":[2], "item":1} | 
| Pre increment | key | n/a | ++i |  {"i":100} |  {"i":101}   | {} | {"key":101} } 
| Post increment | key | n/a | i++ | {"i":100} |  {"i":101}   | {} | {"key":100} } 



**Workflow Lifecycle**

1) New context with a new state map is created after inheriting values from a caller. (Caller will not see any context/state changes from downstream workflow)
2) **data** key is published to the state map with defined workflow.data
2) **params** key is published to state map with the caller parameters
3) Workflow initialization stage executes, applying variables defined in Workflow.Init (input: state, output: state)
4) Tasks Execution 
    1) Task eligibility determination: 
        1) If specified tasks are '*' or empty, all task defined in the workflow will run sequentially, otherwise only specified
        2) Evaluate RunCriteria if specified
    2) Task initialization stage executes, applying variables defined in Task.Init (input: state, output: state)
    
    3) Executes all eligible actions:
        1) Action eligibility determination:
            1) Evaluate RunCriteria if specified
        2) Action initialization stage executes,  applying variables defined in Action.Init (input: state, output: state)
        3) Executing action on specified service
        4) Action post stage executes applying variables defined in Action.Post (input: action.response, output: state)
    4) Task post stage executes, applying variables defined in Task.Post (input: state, output: state)   
5) Workflow post stage executes, applying variables defined in Workflow.Init (input: state, output: workflow.response)




<a name="SystemServices"></a>
## System services


All services are running on the system referred as target and defined as [Resource](https://raw.githubusercontent.com/viant/toolbox/master/url/resource.go)


**Execution services**

The execution service is responsible for opening, managing terminal session, with ability to send command and extract data.

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| exec | open | open SSH session on the target resource. | [OpenSessionRequest](service_exec_session.go#L9) | [OpenSessionResponse](service_exec_session.go#L19) |
| exec | close | close SSH session | [CloseSessionRequest](service_exec_session.go#L24) | [CloseSessionResponse](service_exec_session.goL29) |
| exec | run | executes basic commands | [CommandRequest](service_exec_command.go#L40) | [CommandResponse](service_exec_command_response.go#L15) |
| exec | extract | executes commands with ability to extract data, define error or success state | [ExtractableCommandRequest](service_exec_command.go#L34) | [CommandResponse](service_exec_command_response.go#L15) |



**Daemon service.**

Daemon System service is responsible for managing system daemon services.

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- | 
| daemon | status | check status of system daemon | [DaemonStatusRequest](service_daemon_status.go) | [DaemonInfo](service_daemon_status.go) | 
| daemon | start | start requested system daemon | [DaemonStartRequest](service_daemon_start.go) | [DaemonInfo](service_daemon_status.go) | 
| daemon | stop | stop requested system daemon | [DaemonStopRequest](service_daemon_stop.go) | [DaemonInfo](service_daemon_status.go) | 


**Process service**

Process service is responsible for starting, stopping and checking status of custom application.

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- | 
| process | status | check status of an application | [ProcessStatusRequest](service_process_status.go) | [ProcessStatusResponse](service_process_status.go) | 
| process | start | start provided application | [ProcessStartRequest](service_process_start.go) | [ProcessStartResponse](service_process_start.go) | 
| process | stop | kill requested application | [ProcessStopRequest](service_process_stop.go) | [CommandResponse](exec_command_response.go) | 


**Netowrk service**

<a name="docker"></a>
**Docker service**

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- | 
| docker | run | run requested docker service | [DockerRunRequest](service_docker_run.go) | [DockerContainerInfo](service_docker_container.go#L54) | 
| docker | images | check docker image| [DockerImagesRequest](service_docker_image.go) | [DockerImagesResponse](service_docker_image.go) | 
| docker | stop-images | stop docker containers matching specified images | [DockerStopImagesRequest](service_docker_stop.go) | [DockerStopImagesResponse](service_docker_stop.go) |
| docker | pull | pull requested docker image| [DockerPullRequest](service_docker_pull.go) | [DockerImageInfo](service_docker_image.go) | 
| docker | process | check docker container processes | [DockerContainerCheckRequest](service_docker_container.go) | [DockerContainerCheckResponse](service_docker_container.go) | 
| docker | container-start | start specified docker container | [DockerContainerStartRequest](service_docker_container.go#L19) | [DockerContainerInfo](service_docker_container.go#L54) | 
| docker | container-command | run command within specified docker container | [DockerContainerRunCommandRequest](service_docker_container.go#L39) | [DockerContainerRunCommandResponse](service_docker_container.go#L49) | 
| docker | container-stop | stop specified docker container | [DockerContainerStopRequest](service_docker_container.go#L35) | [DockerContainerInfo](service_docker_container.go#L54) | 
| docker | container-remove | remove specified docker container | [DockerContainerRemoveRequest](service_docker_container.go#L23) | [DockerContainerRemoveResponse](service_docker_container.go#L28) | 
| docker | container-logs | fetch container logs (app stdout/stderr)| [DockerContainerLogsRequest](service_docker_container.go#L63) | [DockerContainerLogsResponse](service_docker_container.go#L69) | 
| docker | inspect | inspect supplied instance name| [DockerInspectRequest](service_docker_inspect.go) | [DockerInspectResponse](service_docker_inspect.go#L12) |
| docker | build | build docker image| [DockerBuildRequest](service_docker_build.go) | [DockerBuildResponse](service_docker_build.go) |
| docker | tag | create a target image that referes to source docker image| [DockerBuildRequest](service_docker_tag.go) | [DockerBuildResponse](service_docker_tag.go) |
| docker | login | store supplied credential for provided repository in local docker store| [DockerLoginRequest](service_docker_login.go) | [DockerLoginResponse](service_docker_login.go) |
| docker | logout | remove credential for supplied repository | [DockerLogoutRequest](service_docker_logout.go) | [DockerLogoutResponse](service_docker_logout.go) |
| docker | push | copy image to supplied repository| [DockerPushRequest](service_docker_push.go) | [DockerPushResponse](service_docker_push.go) |


<a name="storage"></a>
**Storage service**

Storage service represents a local or remote storage to provide unified storage operations.
Remote storage could be any cloud storage i.e google cloud, amazon s3, or simple scp or http.
 

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| storage | copy | copy one or more resources from the source to target destination | [StorgeCopyRequest](service_storage_copy.go) | [StorageCopyResponse](service_storage_copy.go) |
| storage | remove | remove or more resources if exsit | [StorageRemoveRequest](service_storage_remove.go) | [StorageRemoveResponse](service_storage_remove.go) |
| storage | upload | upload content pointed by context state key to target destination. | [StorageUploadRequest](service_storage_copy.go) | [StorageUploadResponse](service_storage_upload.go) |
| storage | download | copy source content into context state key | [StorageDownloadRequest](service_storage_download.go) | [StorageDownloadResponse](service_storage_download.go) |



<a name="CloudAndNetwork"></a>

### Cloud services and Network services


<a name="ec2"></a>

**Amazon Elastic Compute Cloud Service**

Provides ability to call operations on  [EC2 client](https://github.com/aws/aws-sdk-go/tree/master/service/ec2)

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| aws/ec2 | call | run ec2 operation | [EC2CallRequest](service_ec2_call.go) | [EC2CallResponse](service_ec2_call.go)  |

'call' action's method and input are proxied to [EC2 client](https://github.com/aws/aws-sdk-go/tree/master/service/ec2)


<a name="gce"></a>

**Google Compute Engine Service**

Provides ability to call operations on  [*compute.Service client](https://cloud.google.com/compute/docs/reference/latest/)

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| gce | call | run gce operation | [GCECallRequest](service_gce_call.go) | [GCECallResponse](service_gce_call.go)  |

'call' action's service, method and paramters are proxied to [GCE client](https://cloud.google.com/compute/docs/reference/latest/)



**Network service**

Network service is responsible opening tunnel vi SSH between client and target host.
=======
Process service is responsible for starting, stopping and checking status of custom application.

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- | 
| network | tunnel | Tunnel ports between local and remote host | [NetworkTunnelRequest](service_network_tunnel.go) | [NetworkTunnelResponse](service_network_tunnel.go) | 



**SMTP Servoce**

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- | 
| smtp | send | send an email to supplied recipients | [SMTPSendRequest](service_smtp_send.go#L10) | [SMTPSendResponse](service_smtp_send.go#L17) | 




<a name="Buildservices"></a>
## Build and deployment services



**Sdk Service**

Sdk service sets active terminal session with requested sdk version.

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- | 
| sdk | set | set system with requested sdk and version | [SdkSetRequest](service_sdk_set.go) | [SdkSetResponse](service_sdk_set.go) | 


**Version Control Service**

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| version/control | status | run version control check on provided URL | [VcStatusRequest](service_vc_status.go) | [VcInfo](service_vc_info.go)  |
| version/control | checkout | if target directory already  exist with matching origin URL, this action only pulls the latest changes without overriding local ones, otherwise full checkout | [VcCheckoutRequest](service_vc_checkout.go) | [VcInfo](service_vc_info.go)   |
| version/control | commit | commit commits local changes to the version control | [VcCommitRequest](service_vc_commit.go) | [VcInfo](service_vc_info.go)   |
| version/control | pull | retrieve the latest changes from the origin | [VcPullRequest](service_vc_pull.go) | [VcInfo](service_vc_info.go)   |


**Build service**

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| build | load | load BuildMeta for the supplied resource | [BuildLoadMetaRequest](service_build_load.go) | [BuildLoadMetaResponse](service_build_load.go)  |
| build | register | register BuildMeta in service repo | [BuildRegisterMetaRequest](service_build_register.go) | [BuildRegisterMetaResponse](service_build_register.go)  |
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


<a name="selenium"></a>
**Selenium Runner** 

Selenium runner open a web session to run various action on web driver or web elements.

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| selenium | start | start standalone selenium server | [SeleniumServerStartRequest](service_selenium_start.go) | [SeleniumServerStartResponse](service_selenium_start.go) |
| selenium | stop | stop standalone selenium server | [SeleniumServerStopRequest](service_selenium_start.go) | [SeleniumServerStopResponse](service_selenium_stop.go) |
| selenium | open | open a new browser with session id for further testing | [SeleniumOpenSessionRequest](service_selenium_session.go) | [SeleniumOpenSessionResponse](service_selenium_session.go) |
| selenium | close | close browser session | [SeleniumCloseSessionRequest](service_selenium_session.go) | [SeleniumCloseSessionResponse](service_selenium_session.go) |
| selenium | call-driver | call a method on web driver, i.e wb.GET(url)| [SeleniumWebDriverCallRequest](service_selenium_call_web_driver.go) | [SeleniumServiceCallResponse](service_selenium_call_web_driver.go) |
| selenium | call-element | call a method on a web element, i.e. we.Click() | [SeleniumWebElementCallRequest](service_selenium_call_web_element.go) | [SeleniumWebElementCallResponse](service_selenium_call_web_element.go) |
| selenium | run | run set of action on a page | [SeleniumRunRequest](service_selenium_run.go) | [SeleniumRunResponse](service_selenium_run.go) |

call-driver and call-element actions's method and parameters are proxied to stand along selenium server via [selenium client](http://github.com/tebeka/selenium)


selenium run request defines sequence of action, if selector is present then the call method is on [WebDriver](https://github.com/tebeka/selenium/blob/master/selenium.go#L213), 
otherwise [WebElement](https://github.com/tebeka/selenium/blob/master/selenium.go#L370) defined by selector.

[Wait](repeatable.go)  provides ability to wait either some time amount or for certain condition to take place, with regexp to extract data

```json

{
  "SessionID":"$SeleniumSessionID",
  "Actions": [
    {
      "Calls": [
        {
          "Method": "Get",
          "Parameters": [
            "http://play.golang.org/?simple=1"
          ]
        }
      ]
    },
    {
      "Selector": {
        "Value": "#code"
      },
      "Calls": [
        {
          "Method": "Clear"
        },
        {
          "Method": "SendKeys",
          "Parameters": [
            "$code"
          ]
        }
      ]
    },
    {
      "Selector": {
        "Value": "#run"
      },
      "Calls": [
        {
          "Method": "Click"
        }
      ]
    },
    {
      "Selector": {
        "Value": "#output",
        "Key": "output"
      },
      "Calls": [
        {
           "Method": "Text",
           "Wait": {
                    "Repeat": 5,
                    "SleepTimeMs": 100,
                    "ExitCriteria": ":!$value"
           }
        }
      ]
    }
  ]
}
```


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


** Validation expressions **
Generic validation service and log validator, Task or Action RunCritera share underlying [validator](https://github.com/viant/assertly), 
During assertion validator traverses expected data structure to compare it with expected.
If expected keys have not been specified but exists in actual data structure they are being skipped from assertion.


See more validation construct: supported expression, directive and macro go at [Assertly](https://github.com/viant/assertly#validation) 



**Datastore services**

The first action that needs to be run is to register database name with dsc connection config, and optionally init scripts.


| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| dsunit | register | register database connection, and optionally executes init scripts |  [DsUnitRegisterRequest](service_dsunit_register.go) | [DsUnitRegisterResponse](service_dsunit_register.go)  |
| dsunit | mapping |  register virtual mapping between a virtual table and dozen actual tables to simplify setup. |  [DsUnitMappingRequest](service_dsunit_mapping.go) | [DsUnitMappingResponse](service_dsunit_mapping.go)  |
| dsunit | register | register database connection, and optionally executes init scripts |  [DsUnitRegisterRequest](service_dsunit_register.go) | [DsUnitRegisterResponse](service_dsunit_register.go)  |
| dsunit | sequence | takes current sequences for specified tables |  [DsUnitTableSequenceRequest](service_dsunit_sequence.go) | [DsUnitTableSequenceResponse](service_dsunit_sequence.go)  |
| dsunit | sql | executes SQL from supplied URL |  [DsUnitSQLScriptRequest](service_dsunit_sql.go) | [DsUnitSQLScriptResponse](service_dsunit_sql.go)  |
| dsunit | prepare | populates database with setup data |  [DsUnitTablePrepareRequest](service_dsunit_prepare.go) | [DsUnitTablePrepareResponse](service_dsunit_prepare.go)  |
| dsunit | expect | verifies database content with expected data |  [DsUnitTableExpectRequest](service_dsunit_prepare.go) | [DsUnitTableExpectResponse](service_dsunit_prepare.go)  |


To simplify setup/verification data process [DsUnitTableData](service_dsunit_data.go) has been introduce, so that data can be push into state, and then transform to the dsunit expected data with AsTableRecords udf function.


DsUnit uses its own predicate and macro system to perform advanced validation see [Macros And Predicates](../dsunit/docs/)



<a name="Workfowservice"></a>
## Workflow service


**Workflow Service**

Workflow service provide capability to run task, action from any defined workflow.

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| workflow | load | load workflow from provided path | [WorkflowLoadRequest](service_workflow_load.go) | [WorkflowLoadRequest](service_workflow_load.go)  |
| workflow | register | register provide workflow in registry | [WorkflowLoadRequest](service_workflow_register.go) |  |
| workflow | run | run workflow with specified tasks and parameters | [WorkflowRunRequest](service_workflow_run.go) | [WorkflowRunResponse]((service_workflow_run.go) |
| workflow | goto | switche current execution to the specified task on current workflow | [WorkflowGotoRequest](service_workflow_goto.go) | [WorkflowGotoResponse]((service_workflow_goto.go) 
| workflow | switch | run matched  case action or task  | [WorkflowSwitchRequest](service_workflow_switch.go) | [WorkflowSwitchResponse](service_workflow_switch.go) |
| workflow | exit | terminate execution of active workflow (caller) | n/a | n/a |

**Log Service **

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| log | print | print message or error | [LogPrintRequest](service_log.go) | n/a  |

**No Operation Service **


| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| nop | nop | do nothing| [Nop](service_nop.go) | n/a  |
| nop | parrot | return request | [NopParrotRequest](service_nop_parrot.go) | n/a  |
| nop | fail | fail  wokrflow | [NopFailRequest](service_nop_fail.go) | n/a  |
=======



**Predefined workflows**

| Name | Task |Description | 
| --- | --- | --- |
| dokerized_mysql| start | start mysql docker container  |
| dokerized_mysql| stop | stop mysql docker container 
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
 
 
 **Predefined workflow run requests**
 
 | Name | Workflow | 
 | --- | --- | 
 | [tomcat.json](req/tomcat.json) | tomcat | 
 | [aerospike.json](req/aerospike.json)| dockerized_aerospike |
 | [mysql.json](req/mysql.json)| dockerized_mysql |
 | [memcached.json](req/memcached.json)| dockerized_memcached|
   
    

     
<a name="Usage"></a>

## Usage

The following template can be used to run a workflow from a command line 

Note that by default this program will look for run.json

\#endly.go
```go

import (
	"flag"
	"github.com/viant/endly"
	"log"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/viant/asc"//	aerospike
	"time"
)

//TODO add more database drivers import if needed

var workflow = flag.String("workflow", "run.json", "path to workflow run request json file")

func main() {
	flag.Parse()
	runner := endly.NewCliRunner()
	err := runner.Run(*workflow)
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(time.Second)
}


```       


Example of run json

```json
{
  "WorkflowURL": "manager.csv",
  "Name": "manager",
  "PublishParameters":false,
  "EnableLogging":true,
  "LoggingDirectory":"/tmp/myapp/",
  "Tasks":"init,test",
  "Params": {
    "jdkVersion":"1.7",
    "buildGoal": "install",
    "baseSvnUrl":"https://mysvn.com/trunk/ci",
    "buildRoot":"/build",
    "targetHost": "127.0.0.1",
    "targetHostCredential": "${env.HOME}/secret/scp.json",
    "svnCredential": "${env.HOME}/secret/adelphic_svn.json",
    "configURLCredential":"${env.HOME}/secret/scp.json",
    "mysqlCredential": "${env.HOME}/secret/mysql.json",
    "catalinaOpts": "-Xms512m -Xmx1g -XX:MaxPermSize=256m",

    "appRootDirectory":"/use/local",
    "tomcatVersion":"7.0.82",
    "appHost":"127.0.0.1:9880",
    "tomcatForceDeploy":true
  },

  "Filter": {
    "SQLScript":true,
    "PopulateDatastore":true,
    "Sequence": true,
    "RegisterDatastore":true,
    "DataMapping":true,
    "FirstUseCaseFailureOnly":false,
    "OnFailureFilter": {
      "UseCase":true,
      "HttpTrip":true,
      "Assert":true
    }

  }
}

```

See for more filter option: [RunnerReportingFilter](runner_filter.go).
         
         
         
<a name="BestPractice"></a>
## Best Practice

1) Delegate a new workflow request to dedicated req/ folder
2) Variables in  Init, Post should only define state not requests
3) Flag variable as Required or provide a fallback Value
4) Use [Tag Iterators](../neatly) to group similar class of the tests 
5) Since JSON inside tabular cell is not too elegant try to use [Virtual object](../neatly) instead.
6) Organize  workflows and data by  grouping system, datastore, test functionality together. 


Here is an example directory layout.

```text

    manager.csv
        |- system / 
        |      | - system.csv
        |      | - init.json (workflow init variables)
        |      | - req/         
        | - regression /
        |       | - regression.csv
        |       | - init.json (workflow init variables)
        |       | - <use_case_group1> / 1 ... 00X (Tag Iterator)/ <test assets>
        |       | 
        |       | - <use_case_groupN> / 1 ... 00Y (Tag Iterator)/ <test assets>
        | - config /
        |       
        | -  <your app name for related workflow> / //with build, deploy, init, start and stop tasks 
                | <app>.csv
                | init.json 
        
        | - datastore /
                 | - datastore.csv
                 | - init.json
                 | - dictionary /
                 | - schema.ddl
    
```
  
  
  Finally contribute by creating a  pull request with a new common workflows so that other can use them.


## Examples

TODO add some here

         	
<a name="License"></a>
## License

The source code is made available under the terms of the Apache License, Version 2, as stated in the file `LICENSE`.

Individual files may be made available under their own specific license,
all compatible with Apache License, Version 2. Please see individual files for details.


<a name="Credits-and-Acknowledgements"></a>

##  Credits and Acknowledgements

**Library Author:** Adrian Witas

