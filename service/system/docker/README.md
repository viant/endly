# Docker Service

This service is github.com/docker/docker/client.Client proxy 

To check all supported method run
```bash
    endly -s="docker"
```


To check method contract run 
```bash
    endly -s=docker:build
```



See also [docker/ssh](ssh) service

In addition to docker client method proxy methods,  the following were implemented.

### Usage


#### Compacting docker app image

To reduce final image size follow the following steps:

1. Build app binary with all dependencies in the transient container
2. Extract final app binary and static assets/dependencies to final build
3. Build the final image using smallest available base image with prebuild app binary and assets in step 1.

Endly workfow implementing described steps.
* ```endly -r=build```
* [@build.yaml](test/compact/build.yaml)
    ```yaml
    init:
      workingDir: $Pwd()
    pipeline:
      transient:
        build:
          action: docker:build
          path: transient
          noCache: true
          tag:
            image: helloworld
            version: '1.0-transient'
        deploy:
          action: docker:run
          image: 'helloworld:1.0-transient'
          name: helloworld
        extract:
          action: docker:copy
          assets:
            'helloworld:/app/helloworld': ${workingDir}/final/helloworld
        cleanup:
          action: docker:remove
          name: helloworld
          images:
            - 'helloworld:1.0-transient'
      final:
        build:
          action: docker:build
          path: final
          noCache: true
          tag:
            image: helloworld
            version: '1.0'
    ```


#### Docker run

1. Running docker containers in the background
    * docker cli
    ```bash
    docker run --name endly -v /var/run/docker.sock:/var/run/docker.sock -v ~/e2e:/e2e -v ~/e2e/.secret/:/root/.secret/ -p 7722:22  -d endly/endly:latest-ubuntu16.04
    ```
    * equivalent endly workflow [run.yaml](test/run/run.yaml)
    * ```endly -r=run```
    ```yaml
    pipeline:
      run:
        action: docker:run
        image: endly/endly:latest-ubuntu16.04
        name: endly
        # set block to true to wait until container exits
        block: true
        ports:
          7722: 22
        mount:
          /var/run/docker.sock: /var/run/docker.sock
          ~/e2e:/e2e: /e2e
          ~/e2e/.secret: /root/.secret
        env:
          ENDLY: test
    ```
2. Running docker containers with gcr authentication
    * make sure google secrets are placed to ~/.secret/ i.e ~/.secret/gcr.json
    * [@run.yaml](test/run/run_gcr.yaml)
    * ```endly -r=run```
    ```yaml
    pipeline:
      run:
        action: docker:run
        credentials: gcr
        name: dbsync
        mount:
          ~/sync/config/: /config/
          ~/e2e/.secret: /root/.secret
        env:
          ENDLY: test
        image: us.gcr.io/tech-ops-poc/dbsync:1.12
        command: ["./sync", "-s","/config/"]
        ports:
          8082: 8082
    ```

#### Container exec

The Docker service exposes Container Exec APIs via direct bindings. Example flow:

```yaml
pipeline:
  create:
    action: docker:containerExecCreate
    container: my-container
    config:
      cmd: ["/bin/sh", "-c", "echo hello && sleep 1 && echo done"]
      attachStdout: true
      attachStderr: true
  attach:
    action: docker:containerExecAttach
    execID: ${create.Response.ID}
    config:
      detach: false
      tty: false
  start:
    action: docker:containerExecStart
    execID: ${create.Response.ID}
    config:
      detach: false
      tty: false
  inspect:
    action: docker:containerExecInspect
    execID: ${create.Response.ID}
```

All Docker client methods are available as `docker:<methodName in lowerCamelCase>`, for example `docker:containerExecCreate`, `docker:containerExecStart`, `docker:containerExecAttach`, `docker:containerExecInspect`.

#### Docker container status
* docker cli
```bash
    docker ps | grep endly
```
* equivalent endly 
```bash
   endly docker:status name=endly
```

#### Docker build


* endly -r=build
* [@build.yaml](test/build/build.yaml)
```yaml
pipeline:
  build:
    action: docker:build
    path: somePath
    noCache: true
    tag:
      image: helloworld
      version: '1.0'
```
#### Docker login
- authenticate with docker:login 
- ```endly -r=run``` 
- [@run.yaml](test/login/run.yaml)
```yaml
pipeline:
  auth:
    action: docker:login
    credentials: gcr
    repository: us.gcr.io/tech-ops-poc

  run:
    action: docker:run
    name: dbsync
    mount:
      ~/sync/config/: /config/
      ~/e2e/.secret: /root/.secret
    env:
      ENDLY: test
    image: us.gcr.io/tech-ops-poc/dbsync:1.12
    command: ["./sync", "-s","/config/"]
    ports:
      8082: 8082

  unauth:
    action: docker:logout
    repository: us.gcr.io/tech-ops-poc
```

#### Docker copy

1. Copy from container folder
    * docker cp CONTAINTER:/folder/ dest
    * endly -r=copy
    * [@copy.yaml]()
    ```yaml
    pipeline:
      extract:
        action: docker:copy
        assets:
          'container:/folder/': dest
    ```
2. Copy from container folder content (dot)
    * docker cp CONTAINTER:/folder/. dest
    * endly -r=copy
    * [@copy.yaml]()
    ```yaml
    pipeline:
      extract:
        action: docker:copy
        assets:
          'container:/folder/.': dest
    ```
3. Copy to container
    * docker cp source CONTAINTER:/dest/
    * endly -r=copy
    * [@copy.yaml]()
    ```yaml
    pipeline:
      extract:
        action: docker:copy
        assets:
          source: 'container:/folder/'
    ```
4. Multi asset copy
    * endly -r=copy
    * [@copy.yaml]()
    ```yaml
    pipeline:
      extract:
        action: docker:copy
        assets:
          source1: 'container:/folder1/'
          source2: 'container:/folder2/'
          'container:source3': /tmp/folder3
          'container:/app/app': /opt/app/
    ```

#### Docker push
1. hub.docker.com
* ```endly -r=build```
* [@build.yaml](test/push/build_dh.yaml)
    ```yaml
    init:
      version: '1.0'
      image: helloworld
    pipeline:
      build:
        action: docker:build
        path: .
        noCache: true
        tag:
          image: $image
          version: $version
      tag:
        action: docker:tag
        sourceTag:
          image: $image
          version: $version
        targetTag:
          image: $image
          username: myUser
          version: $version
      auth:
        action: docker:login
        repository: index.docker.io/myUser
        credentials: dockerHubmyUser
      pushImage:
        action: docker:push
        tag:
          image: $image
          username: myUser
          version: $version
    ```
2. Google Cloud Registry
* ```endly -r=build```
* [@build.yaml](test/push/build_gcr.yaml)
    ```yaml
    init:
      version: '1.0'
      image: helloworld
    pipeline:
      build:
        action: docker:build
        path: .
        noCache: true
        tag:
          image: $image
          version: $version
      tag:
        action: docker:tag
        sourceTag:
          image: $image
          version: $version
        targetTag:
          image: $image
          registry: us.gcr.io
          username: myUser
          version: $version
      auth:
        action: docker:login
        repository: us.gcr.io/myUser
        credentials: gcr
      pushImage:
        action: docker:push
        tag:
          image: $image
          registry: us.gcr.io
          username: myUser
          version: $version
    ```



### Global parameters

 - APIVersion (optional). If not set, the client negotiates the API version with the daemon. For modern daemons, minimum supported is 1.44.
 
