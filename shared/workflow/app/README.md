### Application workflows

Predefined to easy build and deploy application.

- [Application build deployment](#build)
- [Application build deployment on docker](#deployment)

<a name="build"></a>
## Application build deployment

To enable flexible build and deployment resource delegation the the workflow uses the following resource:

origin - version control origin or some non source control managed location (i.e file system)
target - host resource where endly runs (usually 127.0.0.1 with localhost credentials)
buildTarget  - host resource where app is being built
appTarget - host where app is deployed and runs

If not specified  buildTarget and appTarget use target.
Target is localhost by default.

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

Here is example execution of [build](build/build.csv) and [deploy](deploy/deploy.csv) workflows vi inline pipeline tasks:

```bash
endly -r=app.yaml
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
    action: process:stop-all
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

