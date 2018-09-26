** Docker service**

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- | 
| docker | run | run requested docker service | [RunRequest](service_contract.go) | [ContainerInfo](service_contract.go) | 
| docker | images | check docker image| [ImagesRequest](service_contract.go) | [ImagesResponse](service_contract.go) | 
| docker | stop-images | stop docker containers matching specified images | [StopImagesRequest](service_contract.go) | [StopImagesResponse](service_contract.go) |
| docker | pull | pull requested docker image| [PullRequest](service_contract.go) | [ImageInfo](service_contract.go) | 
| docker | process | check docker container processes | [ContainerCheckRequest](service_contract.go) | [ContainerCheckResponse](service_contract.go) | 
| docker | start | start specified docker container | [ContainerStartRequest](service_contract.go) | [ContainerInfo](service_contract.go) | 
| docker | exec | run command within specified docker container | [ContainerRunCommandRequest](service_contract.go) | [ContainerRunCommandResponse](service_contract.go) | 
| docker | stop | stop specified docker container | [ContainerStopRequest](service_contract.go) | [ContainerInfo](service_contract.go) | 
| docker | remove | remove specified docker container | [ContainerRemoveRequest](service_contract.go) | [ContainerRemoveResponse](service_contract.go) | 
| docker | logs | fetch container logs (app stdout/stderr)| [ContainerLogsRequest](service_contract.go) | [ContainerLogsResponse](service_contract.go) | 
| docker | inspect | inspect supplied instance name| [InspectRequest](service_contract.go) | [InspectResponse](service_contract.go) |
| docker | build | build docker image| [BuildRequest](service_contract.go) | [BuildResponse](service_contract.go) |
| docker | tag | create a target image that referes to source docker image| [BuildRequest](service_contract.go) | [BuildResponse](service_contract.go) |
| docker | login | store supplied credentials for provided repository in local docker store| [LoginRequest](service_contract.go) | [LoginResponse](service_contract.go) |
| docker | logout | remove credentials for supplied repository | [LogoutRequest](service_contract.go) | [LogoutResponse](service_contract.go) |
| docker | push | copy image to supplied repository| [PushRequest](service_contract.go) | [PushResponse](service_contract.go) |
| docker | composeUp | docker compose up| [ComposeUpRequest](service_contract.go) | [ComposeResponse](service_contract.go) |
| docker | comoseDown | docker compose down | [ComposeDownRequest](service_contract.go) | [ComposeResponse](service_contract.go) |



Example of using docker service for building and deploying an app.


```bash
    endly -r=app -t=build,depoy
```


@app.yaml
```yaml
tasks: $tasks
init:
- buildPath = /tmp/build/myapp/
- version = 0.1.0
defaults:
  app: myapp
  version: 0.1.0
  useRegistry: false
pipeline:
  build:
    init:
      action: exec:run
      target: $target
      commands:
      - if [ -e $buildPath ]; then rm -rf $buildPath; fi
      - mkdir -p $buildPath
    checkout:
      action: version/control:checkout
      origin:
        URL: https://github.com/adrianwit/dstransfer
      dest:
        URL: scp://${targetHost}:22/$buildPath
        credentials: localhost
    download:
      action: storage:copy
      source:
        URL: config/Dockerfile
      dest:
        URL: $buildPath
        credentials: localhost
    build-img:
      action: docker:build
      target: $target
      path: $buildPath
      '@tag':
        image: dstransfer
        username: adrianwit
        version: 0.1.0
  stop:
    target: $appTarget
    action: docker:composeDown
    source:
      URL: config/docker-compose.yaml
  deploy:
    target: $appTarget
    action: docker:composeUp
    runInBackground: true
    source:
      URL: config/docker-compose.yaml

```