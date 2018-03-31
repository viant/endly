## Endly Workflow 

- [Introduction](#introduction)
- [Workflow format](#format)
- [Workflow process state](#state)
- [Execution control](#control)
- [Lifecycle](#lifecycle)
- [Best Practise](#best)

<a name="introduction"></a>
### Introduction

**[Workflow](../../model/workflow.go)** an abstraction to define a set actions and tasks.

![diagram](diagram.png)



**Task** an abstraction to logically group one or more action, for example, init,test.

**Action** an abstraction defining a call to a service. 
An action does actual job, like starting service, building and deploying app etc, 

**ActionRequest** an abstraction representing a service request.
        
**ActionResponse** an abstraction representing a service response.

To execute action:
1) workflow service looks up a service by id, in workflow manager registry.
2) workflow service creates a new request for corresponding action on the selected service.
3) Action.Request is expanded with context.State ($variable substitution) to be converted as actual structured service request.
4) Context with its state is passed into every action so that it can be modified for state controlm and future data substitution. 
5) Service executes Run method for provided action to return ServiceResponse 


**[Service](../../service.go)** an abstraction providing set of functionalities triggered by specified action/request.

**State** key/value pair map that is used to mange state during the workflow run. 
The state can be change by providing variable definition.
The workflow content, data structures, can use dollar '$' sign followed by variable name 
to get its expanded to its corresponding state value if the key has been present.
 

<a name="format"></a>
### Format

Endly uses [Neatly](https://github.com/viant/neatly) format to represent a workflow.
Neatly is responsible for converting a tabular document (.csv) into workflow object tree as shown in the [diagram](diagram.png).

Find out more about neatly:
[Neatly introduction](https://github.com/adrianwit/neatly-introduction)



<a name="state"></a>
### Workflow process state

Workflow process uses context.State() to maintain execution state.

**[Variables](variable.go)** an abstraction having capabilities to change a workflow state.

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





    
<a name="control"></a>
### Workflow execution control:
By default, workflow run all specified task, where each task once started executes sequentially all it actions, unless they flag as 'asyn' execution.

Each action can control its execution with

**Action level criteria control**

Each action has the following fields supports [conditional expression](../../criteria) to control workflow execution

1. When: criteria to check if an action is eligible to run
2. Skip: criteria to check if the whole group of actions by TagID can be skipped, continuing execution to next  group
3. Repeater control

    
```go
    type Repeater struct {
    	Extracts     Extracts //textual regexp based data extraction
    	Variables    Variables       //structure data based data extraction
    	Repeat       int             //how many time send this request
    	SleepTimeMs  int             //Sleep time after request send, this only makes sense with repeat option
    	Exit string          //Repeat exit criteria, it uses extracted variable to determine repeat termination 
    }
````
    

        
**Workflow goto task action**
Workflow goto action terminates current task actions execution to start specified current workflow task.`

**Workflow switch action** 
Workflow switch action enables to branch execution based on specified context.state key value. 
Note that switch does not terminate next actions within current task.

**Error handling**
If there is an error during workflow execution, it fails immediately unless OnErrorTask is defined to catch and handle an error.
In addition, error key is placed into the config with the following content:

```go
type WorkflowError struct {
	Error        string
	WorkflowName string
	TaskName     string
	Activity     *WorkflowServiceActivity
}
```


**Finally** 
Workflow also offers DeferTask to execute as the last workflow step in case there is an error or not, for instance, to clean up a resource.


Notify error can be use in conjunction with Workflow.OnTaskError, see below workflow snippet
 
 | Workflow | Name | Tasks | OnErrorTask | | | |
 |---|---|---|---|---|---|---|
 |---|test|%Tasks|onError  | |  | |
 |[]Tasks|Name|Description|Actions| | | |
 | | onError|On error task|%OnError| | | |
 |[]OnError|Description|Service|Action|Request|error|[]receivers|
 | |send error notification | workflow | run | #req/notify_error | $error |	abc@somewehre.com |

 
 <a name="lifecycle"></a>
#### Workflow Lifecycle

1) New context with a new state map is created after inheriting values from a caller. (Caller will not see any state changes from downstream workflow)
2) **data** key is published to the context state with defined workflow.data. Workflow data field would stores complex nested data structure like a setup data.
2) **params** key is published to state map with the caller parameters
3) Workflow initialization stage executes, applying variables defined in Workflow.Pre (input: workflow state, output: workflow state)
4) Tasks Execution 
    1) Task eligibility determination: 
        1) If specified tasks are '*' or empty, all task defined in the workflow will run sequentially, otherwise only specified
        2) Evaluate When if specified
    2) Task initialization stage executes, applying variables defined in Task.Pre (input: workflow  state, output: workflow state)
    
    3) Executes all eligible actions:
        1) Action eligibility determination:
            1) Evaluate When if specified, or Skip for all the actions within the same neatly TagID (tag + Group  + Index + Subpath)
        2) Action initialization stage executes,  applying variables defined in Action.Pre (input: workflow  state, output: workflow  state)
        3) Executing action on specified service
        4) Action post stage executes applying variables defined in Action.Post (input: action.response, output: workflow state)
    4) Task post stage executes, applying variables defined in Task.Post (input: state, output: state)   
5) Workflow post stage executes, applying variables defined in Workflow.Post (input: workflow  state, output: workflow.response)
6) Context state comes with the following build-in/reserved keys:
    * rand - random int64
    * date -  current date formatted as yyyy-MM-dd
    * time - current time formatted as yyyy-MM-dd hh:mm:ss
    * ts - current timestamp formatted  as yyyyMMddhhmmSSS
    * timestamp.yesterday - timestamp in ms
    * timestamp.now - timestamp in ms
    * timestamp.tomorrow - timestamp in ms
    * tmpDir - temp directory
    * uuid.next - generate unique id
    * uuid.get - returns previously generated unique id, or generate new
    *.env.XXX where XXX is the Id of the env variable to return
    * previous - http previous request used for multi request send
    * 
    * all UFD registered functions  
        * [Neatly UDF](https://github.com/viant/neatly/#udf)
        * AsTableRecords udf converting []*DsUnitTableData into map[string][]map[string]interface{} (used by prepare/expect dsunit service), as table record udf provide sequencing and random id generation functionality for supplied data .
	    



         
<a name="best"></a>
## Best Practice

1) Delegate a new workflow request to dedicated req/ folder
2) Variables in  Init, Post should only define state, delegate all variables to var/ folder
3) Flag variable as Required or provide a fallback Value
4) Use [Tag Iterators](https://github.com/viant/neatly#tagiterator) to group similar class of the tests 
5) Since JSON inside a tabular cell is not too elegant, try to use [Virtual object](https://github.com/viant/neatly#vobject) instead.
6) Organize sequential simple tasks into pipeline.
7) Organize functionally cohesive complex tasks into workflows. 


Here is an example directory layout.

```text

      endly
        |- manager.csv
        |- system.yaml              
        |- app.yaml
        |- datastore.yaml
        |
        |- system / 
        | - regression /
        |       | - regression.csv
        |       | - var/init.json (workflow init variables)
        |       | - <use_case_group1> / 1 ... 00X (Tag Iterator)/ <test assets>
        |       | 
        |       | - <use_case_groupN> / 1 ... 00Y (Tag Iterator)/ <test assets>
        | - config /
        |       
        | - datastore /
                 | - dictionary /
                 | - schema.ddl
    
```