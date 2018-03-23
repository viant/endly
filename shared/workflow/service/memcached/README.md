## Memcached workflow


### Start/stop service 


#### _Workflow run_
```bash
# call workflow with task start/stop
endly -w=docker/memcached -t=start
endly -w=docker/memcached -t=stop


#or with parameters
endly -w=docker/memcached -t=start maxMemory=1g
```


#### _Pipeline run_

```bash
endly -p=run
```

@run.yaml
```yaml
pipeline:
  service:
    workflow: docker/memcached
    name: myCache1
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
name: docker/memcached
tasks: start
params:
  name: myCache1
  maxMemory: 1024
```

@run.json
```json
{
  "Name": "docker/memcached",
  "Tasks": "start",
  "Params": {
    "serviceTarget": "$serviceTarget",
    "name": "myCache1",
    "maxMemory":"1024"
  }
}
```


