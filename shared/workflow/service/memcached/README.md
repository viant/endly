## Memcached workflow


### Start/stop service 


#### _Workflow run_
```bash
# call workflow with task start/stop
endly -w=service/memcached -t=start
endly -w=service/memcached -t=stop


#or with parameters
endly -w=service/memcached -t=start maxMemory=1g
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
    workflow: service/memcached
    name: myCache1
  build:
    action: workflow:print
    message: building app ...
```

**Single tasks run**

@run.yaml 
```yaml
name: service/memcached
tasks: start
params:
  name: myCache1
  maxMemory: 1024
```

@run.json
```json
{
  "Name": "service/memcached",
  "Tasks": "start",
  "Params": {
    "serviceTarget": "$serviceTarget",
    "name": "myCache1",
    "maxMemory":"1024"
  }
}
```


