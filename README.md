# Declarative end to end functional testing (endly)

[![Declarative funtional testing for Go.](https://goreportcard.com/badge/github.com/viant/endly)](https://goreportcard.com/report/github.com/viant/endly)
[![GoDoc](https://godoc.org/github.com/viant/endly?status.svg)](https://godoc.org/github.com/viant/endly)

This library is compatible with Go 1.8+

Please refer to [`CHANGELOG.md`](CHANGELOG.md) if you encounter breaking changes.

- [Motivation](#Motivation)
- [Usage](#Usage)
- [Prerequisites](#Prerequisites)
- [Installation](#Installation)
- [API Documentaion](#API-Documentation)
- [Tests](#Tests)
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


## Basic

This testing framework uses neatly format to represent a workflow (TODO add link to neatly)

**Workflow** an abstraction to define a set of task with its action.

**Task** an abstraction to logically group one or more action, for example, init,test.

**Action** an abstraction defining a call to a neatly service action. 
An action does actual job, like starting service, building and deploying app etc, 

**ActionRequest** an abstraction representing a service request.
        
**ActionResponse** an abstraction representing a service response.

**State** key/value pair map that is used to mange state during the workflow run. 
The state can be change by providing variable definition.
The workflow content, data structures, can use dollar '$' sign followed by variable name 
to get its expanded to its corresponding state value if the key has been present.

**Variables** an abstraction defining key to be store in the state map.
Variable has the following attributes
* Name 
Name can be defined as key to be stored in state map or expression with the key
The following expression are supported:


* Value any type value that is used when from value is empty
* From  name of a key state key, or expression with key.    
The following expression are supported:
    * number increments  **++**, for example  counter++, where counter is a key in the state
    * array element shift  **<-**, for example  <-collection, where collection is a key in the state      
    * reference **$** for example $ref, where ref is the key in the state, in this case the value will be 
    evaluated as value stored in key pointed by content of ref variable
    
    
        
* Persist  
* Required

**Wokflow Lifecycle**



## 


## Workflow Services Action

### Workflow Service

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| workflow | load | Loads workflow from provided path |  |  |
| workflow | register | Register provide workflow in registry |  |  |
| workflow | run | run workflow with specified tasks and parameters | [WorkflowRunRequest](service_workflow.go#WorkflowRunRequest)
README.md |  |



## Predefined workflows

 