# Declarative end to end functional testing (endly)

[![Declarative funtional testing for Go.](https://goreportcard.com/badge/github.com/viant/endly)](https://goreportcard.com/report/github.com/viant/endly)
[![GoDoc](https://godoc.org/github.com/viant/endly?status.svg)](https://godoc.org/github.com/viant/endly)

This library is compatible with Go 1.8+

Please refer to [`CHANGELOG.md`](CHANGELOG.md) if you encounter breaking changes.

- [Motivation](#Motivation)
- [Installation](#Installation)
- [Introduction](#Introduction)
- [GettingStarted](#GettingStarted)
- [Services](#Services)
- [Credentials](#Credentail)
- [Unit test](#Unit)
- [Usage](#Usage)
- [Workflow Service](#Workfowservice)
- [Best Practice](#BestPractice)
- [License](#License)
- [Credits and Acknowledgements](#Credits-and-Acknowledgements)




## Motivation

This library was developed to enable simple automated declarative end to end functional testing 
for web application developed in any language.

It addresses all aspect of testing automation namely:
- Local or remote system preparation including all services required by the application.
- Checking out the application code
- Building and deploying the application as a separate process, or in the container.
- Data preparation including RDBMS, or key/value store
- Test use cases with HTTP, REST or selenium runner.
- Verification of responses, data in datastores or log produced.
    

<a name="Installation"></a>
## Installation


[Download the latst endly](https://github.com/viant/endly/releases/)

or build from sources

```text
#optionally set GOPATH directory
export GOPATH=/Projects/go

go get -u github.com/viant/endly


```


<a name="Introduction"></a>
## Introduction

Endly as a comprehensive testing framework automate the following step:

1) System preparation 
    1) Local or remote on cloud
    2) System services initialization. (RDBM, NoSQL, caching or 3rd party API, dockerized services)
    3) Application container. (Docker, Application server, i,e, tomcat, glassfish)
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
    4) Application persisted data verification
    5) Application produced log verification    
4) Cleanup
    1) Data cleanup 
    2) Application shutdown
    3) Application system services shutdown 
    

Endly automate sequence of actions into reusable tasks and workflows. 
It uses tabular [Neatly](https://github.com/viant/neatly) format to represent a workflow.
Neatly is responsible for converting a tabular document (.csv) into workflow object tree as shown below.

![Workflow diagram](workflow_diagram.png)


See more about [workflow and its lifecycle](docs)

A workflow actions invoke endly [services](#Services) to accomplish specific job.




<a name="GettingStarted"></a>
## Getting Started

[Neatly introduction](https://github.com/adrianwit/neatly-introduction)

[Endly introduction](https://github.com/adrianwit/endly-introduction)    



To get you familiar with endly workflows, a few examples of fully functioning applications are included.
You can build, deploy and test them end to end all with endly.

 
1) **Web Service** 
   * [Reporter](example/ws/reporter) - a pivot table report builder.
        - Test with Rest Runner
        - Data Preparation and Validation (mysql)
2) **User Interface**
   * [SSO](example/ui/sso)  - user registration and login application.
        - Test with Selenium Runner
        - Data Preparation and Validation (aersopike)
        - Web Content validation
3) **Extract, Transform and Load (ETL)**
   * [Transformer](example/etl/transformer) - datastore to datastore transformer (i.e. aerospike to mysql)
       - Test with Rest Runner
       - Data Preparation and Validation (aersopike, mysql)
4) **Runtime**  - simple http request event logger
   * [Logger](example/rt/elogger)
       - Test with HTTP Runner
       - Log Validation
       
 

<a name="Services"></a>
## Endly Services

Endly services implement [Service](service.go) interface.

1) **System services**
    - [SSH Executor Service](/system/exec)
    - [Storage Service](/system/storage)
    - [Process Service](/system/process)
    - [Daemon Service](/system/daemon)
    - [Network Service](/system/network)
    - [Docker Service](/system/docker)
2) **Cloud services**
    - [Amazon Elastic Compute Cloud Service](cloud/ec2)
    - [Google Compute Engine Service](cloud/gce)
3) **Build and Deployment Services**
    - [Sdk Service](deployment/sdk)
    - [Version Control Service](deployment/vc)
    - [Build Service](deployment/build)
    - [Deplyment Service](deployment/deploy)
4) **Endpoint Services**
   - [Http Endpoint Service](endpoint/http) 
5) **Runner Services**
   - [Http Runner Service](runner/http) 
   - [REST Runner Service](runner/rest) 
   - [Selenium Runner Service](runner/selenium) 
   - [SMTP Service](runner/smtp)      
6) **Testing Services**
   - [Validator](testing/validator)
   - [Log Validator Service](testing/log)
   - [Datastore Preparation and Validation Service](testing/dsunit)
7) **Workflow service**
    - [Workflow Service](#Workfowservice)
    - [Logger Service](#Workfowservice)
    - [Nop Service](#Workfowservice)
        


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



     
<a name="Credentail"></a>
## Credentials
     
    
Endly on its core uses SSH or other system/cloud service requiring credentials. 
To run system workflow the credentials file/s need to be supplied as various request field.


Endly uses  [Credentail Config](https://github.com/viant/toolbox/blob/master/cred/config.go) 
  * it can store user and blowfish encrypted password generated by "endly -c option".
    * it can store google cloud compatible secret.json fields
  * it can store AWS cloud compatible fields.
  * $HOME/.secret/ directory is used to store endly credentials


Endly service was design in a way to  hide user secrets, for example, whether sudo access is needed,
endly will output **sudo** in the execution event log and screen rather actual password.
     

To generate credentials file to enable endly exec service to run on localhost:

Provide a username and password to login to your box.

```text
mkdir $HOME/.secret
ssh-keygen -b 1024 -t rsa -f id_rsa -P "" -f $HOME/.secret/id_rsa
cat $HOME/.secret/id_rsa.pub >  ~/.ssh/authorized_keys 
chmod u+w authorized_keys

endly -c=localhost -k=~/.secret/id_rsa.pub
```

```
Verify that secret file were created
```text
cat ~/.secret/localhost.json
```     
Now you can use ${env.HOME}./secret/localhost.json as you localhost credential.
     
     
<a name="Usage"></a>

## Usage

In most case scenario you would use **endly** app supplied with [release binary for your platform](https://github.com/viant/endly/releases/).
Alternatively, you can build the latest version of endly with the following command:

```bash

export GOPATH=~/go
go get -u github.com/viant/endly
go get -u github.com/viant/endly/endly

```

endly will be build in the $GOPATH/bin


Make sure its location is on your PATH 


```text

$ endly -h


Usage of endly:
endly [options] [params...]
	params should be key value pair to be supplied as actual workflow parameters
	if -r options is used, original request params may be overriden

where options include:
  -d	enable logging
  -h	print help
  -l string
    	<log directory> (default "logs")
  -p	print neatly workflow as JSON
  -r string
    	<path/url to workflow run request in JSON format>  (default "run.json")
  -t string
    	<task/s to run> (default "*")
  -v	print version (default true)
  -w string
    	<workflow name>  if both -r and -w valid options are specified, -w is ignored (default "manager")
    	

```


When specified workflow or request it can be name of endly [predefined workflow](#predefined_workflows) 
or [request](#predefined_requests).

For instance the following command will print ec2 workflow in JSON format.

```bash

endly -p -w ec2

```

The following command will run predefined ec2 workflow with -w option

```bash

endly -w ec2 -t start awsCredential ~/.secret/aws.json ec2InstanceId i-0ef8d9260eaf47fdd

```

Example of RunRequest JSON

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
    "targetHostCredential": "${env.HOME}/secret/localhost.json",
    "svnCredential": "${env.HOME}/secret/adelphic_svn.json",
    "configURLCredential":"${env.HOME}/secret/localhost.json",
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


See for more filter option: [Filter](cli/runner_filter.go).


In case you have defined you one UDF or have other dependencies you have to build endly binary yourself.
The following template can be used to run a workflow from a command line 


\#endly.go
```go

package main

//import you udf package  or other dependencies here

import "github.com/viant/endly/bootstrap"

func main() {
	bootstrap.Bootstrap()
}

```       
         
    
<a name="Unit"></a>
        
## Unit test 

### Go lang         
         

To integrate endly with unit test, you can use one of the following  
  

**Service action**

With this method, you can run any endly service action directly (including workflow with *endly.WorkflowRunRequest) by providing endly supported request.

This method runs in silent mode.

```go

        manager := endly.NewManager()
    
		response, err := manager.Run(nil, &docker.RunRequest{
            Target: target,
            Image:  "mysql:5.6",
            MappedPort: map[string]string{
                "3306": "3306",
            },
            Env: map[string]string{
                "MYSQL_ROOT_PASSWORD": "**mysql**",
            },
            Mount: map[string]string{
                "/tmp/my.cnf": "/etc/my.cnf",
            },
            Credentials: map[string]string{
                "**mysql**": mySQLcredentialFile,
            },
        })
		if err != nil {
			log.Fatal(err)
		}
		dockerResponse := response.(*docker.RunResponse)
		
```         

**Workfklow**

In this method, a workflow runs with command runner similarly to 'endly' command line.
RunnerReportingOptions settings control stdout/stdin and other workflow details.

```go

    runner := cli.New()
	cli.OnError = func(code int) {}//to supres os.Exit(1) in case of error
	err := runner.Run(&endly.RunRequest{
			WorkflowURL: "action",
			Tasks:       "run",
			Params: map[string]interface{}{
				"service": "logger",
				"action":  "print",
				"request": &endly.PrintRequest{Message: "hello"},
			},
	}, nil)
    if err != nil {
    	log.Fatal(err)
    }

```         




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
 


         
<a name="BestPractice"></a>
## Best Practice

1) Delegate a new workflow request to dedicated req/ folder
2) Variables in  Init, Post should only define state, delegate all variables to var/ folder
3) Flag variable as Required or provide a fallback Value
4) Use [Tag Iterators](https://github.com/viant/neatly#tagiterator) to group similar class of the tests 
5) Since JSON inside a tabular cell is not too elegant, try to use [Virtual object](https://github.com/viant/neatly#vobject) instead.
6) Organize workflows and data by grouping system, datastore, test functionality together. 


Here is an example directory layout.

```text

    manager.csv
        |- system / 
        |      | - system.csv
        |      | - var/init.json (workflow init variables)
        |      | - req/         
        | - regression /
        |       | - regression.csv
        |       | - var/init.json (workflow init variables)
        |       | - <use_case_group1> / 1 ... 00X (Tag Iterator)/ <test assets>
        |       | 
        |       | - <use_case_groupN> / 1 ... 00Y (Tag Iterator)/ <test assets>
        | - config /
        |       
        | -  <your app name for related workflow> / //with build, deploy, init, start and stop tasks 
                | <app>.csv
                | var/init.json 
        
        | - datastore /
                 | - datastore.csv
                 | - var/init.json
                 | - dictionary /
                 | - schema.ddl
    
```
  
  

Finally contribute by creating a  pull request with a new common workflow so that other can use them.


         	
<a name="License"></a>
## License

The source code is made available under the terms of the Apache License, Version 2, as stated in the file `LICENSE`.

Individual files may be made available under their own specific license,
all compatible with Apache License, Version 2. Please see individual files for details.


<a name="Credits-and-Acknowledgements"></a>

##  Credits and Acknowledgements

**Library Author:** Adrian Witas

