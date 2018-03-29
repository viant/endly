## Postgress workflow


###Start/stop service

#### _Workflow run_
```bash
# call workflow with task start/stop     
endly -w=service/pg -t=start

#or with parameters
endly -w=service/pg -t=start name=mydb1

endly -w=service/pg -t=stop
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
    workflow: "service/pg:start"
    name: mydb1
    version: latest
    config: $Pwd(conf/postgress.conf)
  build:
    action: workflow:print
    message: building app ...

```


**Single tasks run**


@run.yaml 
```yaml
name: service/pg
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
  "Name": "service/pg",
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


