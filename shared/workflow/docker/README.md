###Docker workflows

#### Build

| Name | Description |
| ---- | --- |
| [build](build/) | install dependencies and build app |


##### usage:

```bash
endly.go -p=run
```

@run.yaml
```yaml
params:
  app: echo
  secrets:
    localhost: localhsot
  target:
    URL:ssh://127.0.0.1/
    Credentials:localhost
  sdk: go:1.8
pipeline:
  build:
    workflow: docker/build
    origin:
      URL: http://github.com/adrianwit/echo
    upload:
      $Pwd(test/pipeline/build.yaml): /$app/
    commands:
      - apt-get -y install telnet
      - cd /$app
      - go build -o echo
    download:
      /$app: /tmp/$app/
  test:
    action: workflow:print
    message: testing app ...
```
 



#### Services

| Name | Description |
| ---- | --- |
| [aerospike](aerospike/) | start/stop aerospike,  test and wait if it is ready to use |
| [mysql](mysql/) | start/stop mysql,  test and wait till it is ready to use, export, imports data |
| [postgress](pg/) | start/stop postgresSQL, test and wait till it is ready to use |
| [memcached](memcached/) | start/stop memcached |
| [mongoDB](mongo/) | start/stop mongoDB, test and wait till it is ready to use  |
 

##### usage:
 
```bash
# call workflow with task start/stop
endly -w=docker/mysql -t=start
```
 
 