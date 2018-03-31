## Pipeline 

- [Introduction](#introduction)
- [Pipeline format](#format)
- [Pipeline process state](#state)
- [Pipeline Lifecycle](#lifecycle)
- [Shared workflow](#shared)

<a name="introduction"></a>
### Introduction

**[Pipeline](../../model/pipeline.go)** an abstraction defining a simple sequence of tasks.

A task can either execute a [workflow](../workflow) or [service](../service) action.

For instance the following pipeline will execute SSH command (service: exec, action: run).

```bash
endly -r=run
```

@run.yaml
```yaml
pipeline:
  action: exec:run
  target:
    url:  ssh://127.0.0.1/
    credentials: ${env.HOME}/.secret/localhost.json
  commands:
    - mkdir -p /tmp/app/build 
    - chown ${os.user} /tmp/app/build 
```


<name a="format"></a>
### Pipeline format
Pipeline run request can use either JSON or YAML however the latter can be easier choice.


The general pipeline syntax: 

@xxx.yaml
```yaml
params:
  k1:v1
init: var/@init
defaults:
  d1:v1

pipeline:
  node1:
     action: serviceID:action
     requestField1: val1
     requestFieldN: valN
           
  nodeX:
    subNodeX:
      workflow: workflowSelector
      tasks: task selector
      paramsKey1: val1
      paramKeyN: valN
      
    subNodeZ:
       action: serviceID:action
       requestField1: val1
       requestFieldN: valN
      

post: 
  - age = $response.user.age

```
Pipeline execution node is determined by presence of either action or workflow attribute, otherwise
any sub node organization is allowed i.e

```yaml
pipeline:
  service:
    mysql:
      workflow: service/mysql
      tasks: start
    aerospike:
      action: workflow:run
      name: service/aerospike
      tasks: start
  frontend:
    deploy:
      workflow: app/deploy
      sdk: node:8.1
      app: demp-ui
  backend:    
    deploy:
      workflow: app/deploy
      sdk: go:1.9
      app: demo
    
  test:    
```

<a name="state"></a>
### Pipeline process state

Pipeline process usess context.State() to maintain execution state.
All pipeline tasks share the same state.

On the top level initialization and post state modification blocks are supported.
  
Init, Post can be on of the following:
- external reference for **[Variables](./../../model/variable.go)** in JSON or YAML format.
- list of variables assignments i.e.
```yaml
init:
  - target = $params.target
  - app = $params.app
      
```

<a name="lifecycle></a>

### Pipeline lifecycle

1) Parameters are published to the params key in context.state.
2) State initialization block is applied if Init is defined 
3) Defaults key value pairs are appended to _any execution_ node .
4) Pipeline are executed sequentially in the same order as defined in the yaml file.
5) Each execution node publishes action/workflow response result final response Data[ NODE_NAME ] 
6) Post execution modification is applied if Post is defined.

<a name="shared"></a>
### Shared workflow:

Beside endly action pipeline can also use any local or globally [shared workflow](./../../shared/):
Share workflow already provide functionality to prepare system with various service, build, deploy and test an app without reinventing the wheel.