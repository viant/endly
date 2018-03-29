## Aerospike workflow


### Start/stop service 


#### _Workflow run_
```bash
# call workflow with task start/stop
endly -w=service/aerospike -t=start
endly -w=service/aerospike -t=stop


#or with parameters
endly -w=service/aerospike -t=start name=myDB
```


#### _Workflow with parameters_

```bash
endly -r=run
```

**Pipeline multi tasks run**

@run.yaml
```yaml
pipeline:
  service:
    workflow: service/aerospike:start
    name: mydb1
    version: latest
  build:
    action: workflow:print
    message: building app ...
```


**Single task run**

@run.yaml 
```yaml
name: service/aerospike
tasks: start
params:
  name: mydb1
  version: latest
  config: $Pwd(conf/aerospike.cnf)
  env:
    NAMESPACE: aerospike-demo
```

@run.json
```json
{
  "Name": "service/aerospike",
  "Tasks": "$tasks",
  "Params": {
    "target": "$target",
    "serviceTarget": "$serviceTarget",
    "config": "$config",
    "name": "$instance",
    "version": "$version"
  }
}
```
