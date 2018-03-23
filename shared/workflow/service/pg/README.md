## Postgress workflow


###Start/stop service

#### _Workflow run_
```bash
# call workflow with task start/stop     
endly -w=docker/pg -t=start

#or with parameters
endly -w=docker/pg -t=start name=mydb1

endly -w=docker/pg -t=stop
```


#### _Pipeing workflows/actions_

```bash
endly -p=run
```

@run.yaml
```yaml
pipeline:
  service:
    workflow: "docker/pg:start"
    name: mydb1
    version: latest
    config: $Pwd(conf/postgress.conf)
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
name: docker/pg
tasks: start
params:
  name: mydb1
  credentials: secret.json
  
```

**Default credential:** 
Default credentials file uses the following: root/dev


@run.json
```json
{
  "Name": "docker/pg",
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


