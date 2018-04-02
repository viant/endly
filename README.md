# Declarative end to end functional testing (endly)

[![Declarative funtional testing for Go.](https://goreportcard.com/badge/github.com/viant/endly)](https://goreportcard.com/report/github.com/viant/endly)
[![GoDoc](https://godoc.org/github.com/viant/endly?status.svg)](https://godoc.org/github.com/viant/endly)

This library is compatible with Go 1.8+

Please refer to [`CHANGELOG.md`](CHANGELOG.md) if you encounter breaking changes.

- [Motivation](#Motivation)
- [Installation](#Installation)
- [Introduction](#Introduction)
- [GettingStarted](#GettingStarted)
- [Documentation](#Documentation)
- [License](#License)
- [Credits and Acknowledgements](#Credits-and-Acknowledgements)




## Motivation

This library was developed in go lang to enable simple automated declarative end to end functional testing 
for web application developed in any language.

It addresses all aspect of testing automation namely:
- Local or remote system preparation including all services required by the application.
- Checking out the application code
- Building and deploying the application as a separate process, or in the container.
- Data preparation including RDBMS, or key/value store
- Test use cases with HTTP, REST or selenium runner.
- Verification of responses, data in datastores or log produced.
    

<a name="Installation"></a>
## Installation

1) [Download latest binary](https://github.com/viant/endly/releases/)
    ```bash
     tar -xvzf endly_xxx.tar.gz
     cp endly /usr/local/bin
     endly -h
     endly -v

    ```
 

2) Build from source
   a) install go 1.9+
   b) run the following commands:
   ```bash
   mkdir -p ~/go
   export GOPATH=~/go
   go get -u  github.com/viant/endly
   go get -u  github.com/viant/endly/endly
   cd $GOPATH/src/github.com/viant/endly/endly
   go build endly.go
   cp endly /usr/local/bin
   ```
3) Custom build, in case you need additional drivers, dependencies or UDF, added necessary imports:

@endly.go
```go

package main

//import your udf package  or other dependencies here

import "github.com/viant/endly/bootstrap"

func main() {
	bootstrap.Bootstrap()
}

```       


<a name="Introduction"></a>
## Introduction

Endly as a comprehensive testing framework automate the following step:

1) System preparation 
    1) Local or remote on cloud
    2) System services initialization. (RDBM, NoSQL, caching or 3rd party API, dockerized services)
    3) Application container. (Docker, Application server, i,e, tomcat, glassfish)
2) Application build and deployment
    1) Application code checkout.
    2) Application build
    3) Application deployment
3) Testing
    1) Preparing test data
    2) Actual application testing
        1) Http runner
        2) Reset runner
        3) Selenium runner
    3) Application output verification
    4) Application persisted data verification
    5) Application produced log verification    
4) Cleanup
    1) Data cleanup 
    2) Application shutdown
    3) Application system services shutdown 
    

<a name="GettingStarted"></a>
## Getting Started

Endly automate sequence of actions into reusable tasks and workflows or inline pipeline tasks. 

**a) System preparation**

For instance: the following define inline pipeline tasks to prepare app system services:

@system.yaml
```yaml
tasks: $tasks
defaults:
  target: $serviceTarget
pipeline:
  destroy:
    stop-images:
      action: docker:stop-images
      images:
        - mysql
        - aerospike
  init:
    services:
      mysql:
        workflow: "service/mysql:start"
        name: mydb3
        version: $mysqlVersion
        credentials: $mysqlCredentials
        config: config/my.cnf
      aerospike:
        workflow: "service/aerospike:start"
        name: mydb4
        config: config/aerospike.conf
```


**b) Application build and deployment** 

For instance: the following  define inline pipeline tasks to build and deploy a test app:
(you can easily build an app for standalone mode or in and for docker container)

@app.yaml
```yaml
tasks: $tasks
defaults:
  app: myApp
  sdk: go:1.8

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


**c) Datastore creation**

For instance: the following  define inline pipeline tasks to create/populare mysql and aerospike database/dataset:

@datastore.yaml

```yaml
pipeline:
  create-db:
    db3:
      action: dsunit:init
      scripts:
        - URL: datastore/db3/schema.ddl
      datastore: db3
      recreate: true
      config:
        driverName: mysql
        descriptor: "[username]:[password]@tcp(127.0.0.1:3306)/[dbname]?parseTime=true"
        credentials: $mysqlCredentials
      admin:
        datastore: mysql
        config:
          driverName: mysql
          descriptor: "[username]:[password]@tcp(127.0.0.1:3306)/[dbname]?parseTime=true"
          credentials: $mysqlCredentials
    db4:
      action: dsunit:init
      datastore: db4
      recreate: true
      config:
        driverName: aerospike
        descriptor: "tcp([host]:3000)/[namespace]"
        parameters:
          dbname: db4
          namespace: db4
          host: $serviceHost
          port: 3000
  populate:
    db3:
      action: dsunit:prepare
      datastore: db3
      URL: datastore/db3/dictionary
    db4:
      action: dsunit:prepare
      datastore: db4
      URL: datastore/db4/data
```

**d) Testing**

For instance: the following  define inline pipeline tasks to run test with selenium runner:


@test.yaml

```yaml
defaults:
  target:
     URL: ssh://127.0.0.1/
     credentials: localhost
pipeline:
  init:
    action: selenium:start
    version: 3.4.0
    port: 8085
    sdk: jdk
    sdkVersion: 1.8
  test:
    action: selenium:run
    browser: firefox
    remoteSelenium:
      URL: http://127.0.0.1:8085
    commands:
      - get(http://play.golang.org/?simple=1)
      - (#code).clear
      - (#code).sendKeys(package main

          import "fmt"

          func main() {
              fmt.Println("Hello Endly!")
          }
        )
      - (#run).click
      - command: output = (#output).text
        exit: $output.Text:/Endly/
        sleepTimeMs: 1000
        repeat: 10
      - close
    expect:
      output:
        Text: /Hello Endly!/
```



To show _Endly_ in action, a few examples of fully functioning applications are included.
You can build, deploy and test them end to end all with endly.

 
1) **Web Service** 
   * [Reporter](example/ws/reporter) - a pivot table report builder.
        - Test with Rest Runner
        - Data Preparation and Validation (mysql)
2) **User Interface**
   * [SSO](example/ui/sso)  - user registration and login application.
        - Test with Selenium Runner
        - Data Preparation and Validation (aersopike)
        - Web Content validation
3) **Extract, Transform and Load (ETL)**
   * [Transformer](example/etl/transformer) - datastore to datastore transformer (i.e. aerospike to mysql)
       - Test with Rest Runner
       - Data Preparation and Validation (aersopike, mysql)
4) **Runtime**  - simple http request event logger
   * [Logger](example/rt/elogger)
       - Test with HTTP Runner
       - Log Validation
5) **Automation** - simple 3rd party echo app
    * [Echo](example/au/echo)
        - Build 3rd party application binary in docker container
        - Build application docker image
        - Optionally publish app image to the docker registry
        - Deploy app to docker container
        - Test an app with REST and HTTP runner


<a name="Documentation"></a>

## Documentation
- [Secret/Credential](doc/secrets)
- [Service](doc/service)
- [Usage](doc/usage)
- [Pipeline](doc/pipeline)
- [Workflow](doc/workflow)



## External resources

[Endly introduction](https://github.com/adrianwit/endly-introduction)    

         	
<a name="License"></a>
## License

The source code is made available under the terms of the Apache License, Version 2, as stated in the file `LICENSE`.

Individual files may be made available under their own specific license,
all compatible with Apache License, Version 2. Please see individual files for details.



<a name="Credits-and-Acknowledgements"></a>

##  Credits and Acknowledgements

**Library Author:** Adrian Witas

