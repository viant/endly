## mongo workflow


### Start/stop service

#### _Workflow run_
```bash
# call workflow with task start/stop
endly -w=docker/mongo -t=start
endly -w=docker/mongo -t=stop


#or with parameters
endly -w=docker/mongo -t=start name=myDB
```

#### _Pipeline run_

```bash
endly -p=run
```

@run.yaml
```yaml
pipeline:
  service:
    workflow: docker/mongo:start
    name: mydb1
    version: latest
  build:
    action: workflow:print
    message: building app ...
```


#### _Workflow run with custom parameters_
 
 
```bash      
endly -r=run.yaml
```


@run.yaml 
```yaml
name: docker/mongo
tasks: start
params:
  name: mydb1
  version: latest
```
