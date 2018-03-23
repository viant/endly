## Aerospike workflow


### Start/stop service 


#### _Workflow run_
```bash
# call workflow with task start/stop
endly -w=docker/aerospike -t=start
endly -w=docker/aerospike -t=stop


#or with parameters
endly -w=docker/aerospike -t=start name=myDB
```


#### _Pipeline run_

```bash
endly -p=run
```

@run.yaml
```yaml
pipeline:
  service:
    workflow: docker/aerospike:start
    name: mydb1
    version: latest
  build:
    action: workflow:print
    message: building app ...
```

#### _Workflow run with custom parameters_
 
 
```bash      
endly -r=run.yaml
endly -r=run.json
```


@run.yaml 
```yaml
name: docker/aerospike
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
  "Name": "docker/aerospike",
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


