## mongo workflow


### Start/stop service

#### _Workflow run_
```bash
# call workflow with task start/stop
endly -w=service/mongo -t=start
endly -w=service/mongo -t=stop


#or with parameters
endly -w=service/mongo -t=start name=myDB
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
    workflow: service/mongo:start
    name: mydb1
    version: latest
  build:
    action: workflow:print
    message: building app ...
```


**Single tasks run**

@run.yaml 
```yaml
name: service/mongo
tasks: start
params:
  name: mydb1
  version: latest
```
