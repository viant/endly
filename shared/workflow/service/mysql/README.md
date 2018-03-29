## MySQL workflow


###Export and import


```bash
endly -w=docker/mysql -t=export dump='/tmp/out.sql'
endly -w=docker/mysql -t=import dump='/tmp/out.sql'
```


###Start/stop service

#### _Workflow run_
```bash
# call workflow with task start/stop
endly -w=docker/mysql -t=start

#or with parameters
endly -w=docker/mysql -t=start name=mydb1

endly -w=docker/mysql -t=stop
```



#### _Pipeing workflows/actions_

```bash
endly -p=run
```

@run.yaml
```yaml
pipeline:
  service:
    workflow: "docker/mysql:start"
    name: mydb1
    version: latest
    config: $Pwd(conf/my.cnf)
  build:
    action: workflow:print
    message: building app ...

```


**Single tasks run**


```yaml
name: docker/mysql
tasks: start
params:
  name: mydb1
  version: 5.6
  credentials: secret.json
  
```

**Default credential:** 
Default credentials file uses the following: root/dev


@run.json
```json
{
  "Name": "docker/mysql",
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


