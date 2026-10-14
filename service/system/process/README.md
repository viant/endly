# Process service


Process service is responsible for starting, stopping and checking the status of a custom application.
It uses SSH service.



### Usage

1. Starting process
```endly app.yaml ```
[app.yaml](usage/app.yaml)
```yaml
init:
  appPath: $Pwd()/my-app
pipeline:
  setSdk:
    action: sdk:set
    sdk: node:12
  build:
    action: exec:run
    checkError: true
    commands:
      - cd $appPath
      - npm install
      - npm test
  stop:
    action: process:stop
    input: react-scripts/scripts/start.js

  start:
    action: process:start
    directory: $appPath/
    watch: true
    immuneToHangups: true
    command: npm start
```

2. Stopping process

###

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- | 
| process | status | check status of an application | [StatusRequest](service_contract.go) | [StatusResponse](service_contract.go) | 
| process | start | start provided application | [StartRequest](service_contract.go) | [StartResponse](service_contract.go) | 
| process | stop | kill requested application | [StopRequest](service_contract.go) | [RunResponse](../exec/service_contract.go) | 


For POSIX processes whose arguments contain DSNs, spaces or shell metacharacters,
set `quoteArguments: true` on `process:start`. Command and argument strings are
then passed as literal shell arguments; the directory and output path are quoted
as well. The default remains false for existing workflows that intentionally
compose shell command text.
