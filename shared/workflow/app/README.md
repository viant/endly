### Application workflows

Predefined to easy build and deploy application.

- [Application build deployment](#build)
- [Application build deployment on docker](#deployment)


**Build resources:**

To enable flexible build and deployment resource delegation, build workflows use the following resource:

origin - version control origin or some non source control managed location (i.e file system)
target - host resource where endly runs (usually 127.0.0.1 with localhost credentials)
buildTarget  - host resource where app is being built
appTarget - host where app is deployed and runs

If not specified  buildTarget and appTarget use target.
Target is localhost by default.


<a name="build"></a>
## Application build deployment


Here are the basic step with [build](build/build.csv) and [deploy](deploy/deploy.csv) workflows.

1) Build
- create $buildPath (default: $buildTarget:/tmp/${app}/build/)
- checkout code (from source control or local resource ) to build host
- set SDK
- upload additional assets to $buildPath (default: $buildTarget:/tmp/${app}/build/)
- download build commands
- downloading build app and other assets to $target release path  (default: $target:/tmp/${app}/release/)
2) Deploy
- create appPath (default: $appTarget:/opt/${app}/)
- upload application from $releasePath to $appTarget
- stop app (if previous instance is still running)
- start app

Here is example execution of [build](build/build.csv) and [deploy](deploy/deploy.csv) workflows vi inline workflow:

```bash
endly -r=app.yaml -t='*'
```


@app.yaml
```yaml
tasks: $tasks
defaults:
  app: $app
  sdk: $sdk
  target: $target
  buildTarget: $buildTarget
  appTarget: $appTarget

pipeline:

  build:
    workflow: app/build
    origin:
      URL: ./../
    commands:
      - cd $buildPath/app
      - go get -u .
      - go build -o $app
      - chmod +x $app
    download:
      /$buildPath/app/${app}: $releasePath
      /$buildPath/endly/config/config.json: $releasePath

  deploy:
    workflow: app/deploy
    init:
      - mkdir -p $appPath
      - mkdir -p $appPath/config
      - chown -R ${os.user} $appPath
    upload:
      ${releasePath}/${app}: $appPath
      ${releasePath}/config.json: $appPath
    commands:
      - echo 'deployed'

  stop:
    action: process:stop
    input: ${app}

  start:
    action: process:start
    directory: $appPath
    immuneToHangups: true
    command: ./${app}
    arguments:
      - "-config"
      - "config.json"

```

<a name="deployment"></a>
## Application build deployment on docker



Here are the basic step with [build](docker/build/build.csv) and [deploy](docker/deploy/deploy.csv) workflows.

1) Build
- launches ubuntu docker with SSH service enabled as buildTarget
- checkout code (from source control or local resource ) to build host
- set SDK
- upload additional assets to $buildPath (default: $buildTarget:/tmp/${app}/build/)
- download build commands
- downloading build app and other assets to $target release path  (default: $target:/tmp/${app}/release/)
- if dockerfile is specified or config/Dockerfile is present then it build app docker image
- tag image and deploy it docker registry if useRegistry is set to true

2) Deploy
- optionally login to docker registry (if useRegistry flag is set)
- stop app docker container (if previous instance is still running)
- run docker



```yaml
tasks: $tasks
params:
  app: echo
  version: 0.1
  sdk: go:1.8
  registryUsername: endly
  registry: index.docker.io/endly
  useRegistry: false
defaults:
  app: $app
  version: $version
  sdk: $sdk
pipeline:
  build:
    workflow: app/docker/build
    origin:
      URL: https://github.com/adrianwit/echo
    commands:
      - apt-get -y install telnet
      - cd $buildPath/app
      - go get -u .
      - export CGO_ENABLED=0
      - go build -o $app
      - chmod +x $app
    download:
      /$buildPath/${app}: $releasePath
  stop:
    action: docker:stop
    images:
    - testApp
  deploy:
    workflow: app/docker/deploy
    name: testApp
    ports:
      "8080": "8080"

```