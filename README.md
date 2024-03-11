# Declarative end to end functional testing (endly)

[![Declarative funtional testing for Go.](https://goreportcard.com/badge/github.com/viant/endly)](https://goreportcard.com/report/github.com/viant/endly)
[![GoDoc](https://godoc.org/github.com/viant/endly?status.svg)](https://godoc.org/github.com/viant/endly)

This library is compatible with Go 1.12+

Please refer to [`CHANGELOG.md`](CHANGELOG.md) if you encounter breaking changes.

- [Motivation](#motivation)
- [Introduction](#introduction)
- [Getting Started](#getting-started)
- [Documentation](#documentation)
- [License](#license)

- [Credits and Acknowledgements](#credits-and-acknowledgements)


## Motivation

Endly is comprehensive workflow based automation and end-to-end (E2E) testing tool designed to simulate a production environment as closely as possible. 
This includes the full spectrum of network communications, user interactions, data storage, and other dependencies. 
By doing so, it aims to ensure that systems are thoroughly tested under conditions that mimic real-world operations, 
helping to identify and address potential issues before deployment.

## Introduction

Endly is a highly versatile automation and orchestration platform that provides a wide range of services to support various aspects of software development, 
testing, deployment, and operations. 
Below is a summary of the types of services Endly can orchestrate, grouped by their primary functionality:

### Platform, Infrastructure and Cloud Providers:
#### Docker:
Provides services for managing Docker containers and executing commands over SSH within Docker environments, enhancing container management and deployment.
#### AWS Services: 
Offers orchestration for numerous AWS services, including API Gateway, CloudWatch, DynamoDB, EC2, IAM, Kinesis, KMS, Lambda, RDS, S3, SES, SNS, SQS, and SSM. These services enable management and automation of AWS resources, monitoring, notification, and security.
#### GCP Services: 
Supports Google Cloud Platform resources such as BigQuery, Cloud Functions, Cloud Scheduler, Compute Engine, GKE (Google Kubernetes Engine), KMS, Pub/Sub, Cloud Run, and Cloud Storage. These services are essential for managing Google Cloud resources, data analysis, event-driven computing, and storage.
#### Kubernetes: 
Automates tasks within Kubernetes clusters, covering apps, autoscaling, batch processing, core services, extensions, networking, policy management, RBAC (Role-Based Access Control), settings, and storage. This facilitates the deployment, scaling, and management of containerized applications.

### Environment/System Management
#### Exec: 
This service is central to executing shell commands, allowing for automation of tasks that interact directly with the operating system. This capability is essential for setting up environments, running scripts, and performing system-level operations, thereby serving as a foundation for environment and system management within Endly's orchestration capabilities.
#### Process: 
Manages processes or daemons on the system, enabling control over the lifecycle of various applications or services running in the background. This service is key for ensuring that necessary services are operational during testing or deployment, and for automating start, stop, and restart operations of system services as part of environment setup and maintenance.
#### Storage: 
Facilitates the management of file-based assets, including uploading, downloading, and managing files. This service is crucial for handling configuration files, test data, and other file-based resources needed throughout the automation, testing, and deployment processes. It supports the simulation of real-world environments by ensuring the correct setup of file systems and data storage scenarios.
#### Secret: 
Manages safe access to secrets, such as passwords and API keys, crucial for maintaining security in automated processes.
#### Development and Deployment
Build and Deployment: Includes services for building software and deploying applications, encompassing general build and deployment tasks, version control with Git, and specific deployment strategies.

### Database and Data Management
#### DSUnit: 
Facilitates database testing, supporting the setup and teardown of database states for testing purposes.

### Testing and Integration
#### HTTP/REST: 
Provides tools for testing and interacting with HTTP endpoints and RESTful APIs. This service is instrumental in API testing, allowing for the automation of requests, response validation, and the simulation of various API usage scenarios. It supports both the verification of external services and the testing of application interfaces, making it a vital component for ensuring the functionality and reliability of web services and applications.
#### HTTP/Endpoint: 
This service extends Endly's capabilities into the realm of HTTP communication testing by allowing users to listen to existing HTTP interactions, record them, and subsequently mock these interactions for the purpose of testing. This functionality is particularly useful for simulating third-party HTTP integrations without the need for the third-party services to be actively involved in the testing process. By capturing and replicating the behavior of external HTTP services, it enables developers to conduct thorough testing of application integrations in a controlled, predictable environment. This approach ensures that applications can gracefully handle external HTTP requests and responses, facilitating the validation of integration points with external APIs and services. The ability to mock external HTTP interactions is invaluable for continuous integration and testing workflows, where external dependencies must be accurately simulated to verify application functionality and resilience.
#### Selenium: 
Supports browser-based testing and automation, essential for web application testing.
### Validator: 
Provides validation services, including log validation, to ensure that applications behave as expected.
Communication and Messaging
#### SMTP, Slack: 
Services for sending emails and Slack messages, enabling notifications and alerts as part of the automation workflows.
## Getting Started


##### Installation
  - [Install/Download](doc/installation)
  - [Endly docker image](docker/)



##### Examples:


**a) Infrastructure/Environment preparation**

For instance: the following define inline workflow to prepare app system services:

@system.yaml
```yaml
tasks: $tasks
defaults:
  target: $serviceTarget
pipeline:
  destroy:
    stop-images:
      action: docker:stop
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

For instance: the following  define inline workflow to build and deploy a test app:
(you can easily build an app for standalone mode or in and for docker container)


**With Dockerfile build file and docker compose**

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
    action: docker/ssh:composeDown
    source:
      URL: config/docker-compose.yaml
  deploy:
    target: $appTarget
    action: docker/ssh:composeUp
    runInBackground: true
    source:
      URL: config/docker-compose.yaml

```

**As Standalone app** (with predefined shared workflow)


@app.yaml
```yaml
init:
  buildTarget:
    URL: scp://127.0.0.1/tmp/build/myApp/
    credentials: localhost
  appTarget:
    URL: scp://127.0.0.1/opt/myApp/
    credentials: localhost
  target:
    URL: scp://127.0.0.1/
    credentials: localhost
defaults:
  target: $target

pipeline:

  build:
    checkout:
      action: version/control:checkout
      origin:
        URL: ./../ 
        #or https://github.com/myRepo/myApp
      dest: $buildTarget
    set-sdk:
      action: sdk:set
      sdk: go:1.17
    build-app:
      action: exec:run
      commands:
        - cd /tmp/build/myApp/app
        - export GO111MODULE=on
        - go build myApp.go
        - chmod +x myApp
    deploy:
      mkdir:
        action: exec:run
        commands:
          - sudo rm -rf /opt/myApp/
          - sudo mkdir -p /opt/myApp
          - sudo chown -R ${os.user} /opt/myApp

      install:
        action: storage:copy
        source: $buildTarget
        dest: $appTarget
        expand: true
        assets:
          app/myApp: myApp
          config/config.json: config.json

  stop:
    action: process:stop
    input: myApp

  start:
    action: process:start
    directory: /opt/myApp
    immuneToHangups: true
    command: ./myApp
    arguments:
      - "-config"
      - "config.json"

```

**c) Datastore/database creation**

For instance: the following  define inline workflow to create/populare mysql and aerospike database/dataset:

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

```bash
endly -r=datastore
```

**d) Creating setup / verification dataset from existing datastore**


For instance: the following  define inline workflow to create setup dataset SQL based from on existing database

@freeze.yaml
```yaml
pipeline:
  db1:
    register:
      action: dsunit:register
      datastore: db1
      config:
        driverName: bigquery
        credentials: bq
        parameters:
          datasetId: adlogs

    reverse:
      takeSchemaSnapshot:
        action: dsunit:dump
        datastore: db1
        # leave empty for all tables
        tables:
          - raw_logs
        #optionally target for target vendor if different that source  
        target: mysql 
        destURL: schema.sql
        
      takeDataSnapshot:
        action: dsunit:freeze
        datastore: db1
        destURL: db1/prepare/raw_logs.json
        omitEmpty: true
        ignore:
          - request.postBody
        replace:
          request.timestamp: $$ts
        sql:  SELECT request, meta, fee
                FROM raw_logs 
                WHERE requests.sessionID IN(x, y, z)
```

```bash
endly -r=freeze
```


**e) Comparing SQL based data sets**

```bash
endly -r=compare
```

@compare.yaml
```yaml
pipeline:
  register:
    verticadb:
      action: dsunit:register
      datastore: db1
      config:
        driverName: odbc
        descriptor: driver=Vertica;Database=[database];ServerName=[server];port=5433;user=[username];password=[password]
        credentials: db1
        parameters:
          database: db1
          server: x.y.z.a
          TIMEZONE: UTC
    bigquerydb:
      action: dsunit:register
      datastore: db2
      config:
        driverName: bigquery
        credentials: db2
        parameters:
          datasetId: db2
  compare:
    action: dsunit:compare
    maxRowDiscrepancy: 10
    ignore:
      - field10
      - fieldN
    directives:
      "@numericPrecisionPoint@": 7
      "@coalesceWithZero@": true
      "@caseSensitive@": false
    
    source1:
      datastore: db1
      SQL: SELECT * 
           FROM db1.mytable 
           WHERE DATE(ts) BETWEEN '2018-12-01' AND '2018-12-02' 
           ORDER BY 1

    source2:
      datastore: db2
      SQL: SELECT *
           FROM db2.mytable
           WHERE DATE(ts) BETWEEN '2018-12-01' AND '2018-12-02'
           ORDER BY 1
```


**f) Testing**

For instance: the following  define inline workflow to run test with selenium runner:


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

```bash
endly -r=test
```


**g) Stress testing:**

The following  define inline workflow that loads request and desired responses from data folder for stress testing.

@load.yaml
```yaml
init:
  testEndpoint: z.myendoint.com
pipeline:
  test:
    tag: StressTest
    data:
      []Requests: '@data/*request.json'
      []Responses: '@data/*response.json'
    range: '1..1'
    template:
      info:
        action: print
        message: starting load testing 
      load:
        action: 'http/runner:load'
        threadCount: 3
        '@repeat': 100
        requests: $data.Requests
        expect:
          Responses: $data.Responses
      load-info:
        action: print
        message: 'QPS: $load.QPS: Response: min: $load.MinResponseTimeInMs ms, avg: $load.AvgResponseTimeInMs ms max: $load.MaxResponseTimeInMs ms'

```

Where data folder contains http request and desired responses i.e 

@data/XXX_request.json
```json
{
  "Method":"get",
  "URL":"http://${testEndpoint}/bg/?pixid=123"
}
```

@data/XXX_response.json
```json
{
  "Code":200,
  "Body":"/some expected fragement/"
}
```


```bash
endly -r=load
```



**h) Serverless e2e testing with cloud function**


@test.yaml
```yaml
defaults:
  credentials: am
pipeline:
  deploy:
    action: gcp/cloudfunctions:deploy
    '@name': HelloWorld
    entryPoint: HelloWorldFn
    runtime: go111
    source:
      URL: test/
  test:
    action: gcp/cloudfunctions:call
    logging: false
    '@name': HelloWorld
    data:
      from: Endly
  info:
    action: print
    message: $test.Result
  assert:
    action: validator:assert
    expect: /Endly/
    actual: $test.Result
  undeploy:
    action: gcp/cloudfunctions:delete
    '@name': HelloWorld

```

**i) Serverless e2e testing with lambda function**

@test.yaml
```yaml
init:
  functionRole: lambda-loginfo-executor
  functionName: LoginfoFn
  codeZip: ${appPath}/loginfo/app/loginfo.zip
  privilegePolicy: ${parent.path}/privilege-policy.json
pipeline:
  deploy:
    build:
      action: exec:run
      target: $target
      errors:
        - ERROR
      commands:
        - cd ${appPath}loginfo/app
        - unset GOPATH
        - export GOOS=linux
        - export GOARCH=amd64
        - go build -o loginfo
        - zip -j loginfo.zip loginfo

    setupFunction:
      action: aws/lambda:deploy
      credentials: $awsCredentials
      functionname: $functionName
      runtime:  go1.x
      handler: loginfo
      code:
        zipfile: $LoadBinary(${codeZip})
      rolename: lambda-loginfo-executor
      define:
        - policyname: s3-mye2e-bucket-role
          policydocument: $Cat('${privilegePolicy}')
      attach:
        - policyarn: arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole

    setupAPI:
      action: aws/apigateway:deployAPI
      credentials: aws
      '@name': loginfoAPI
      resources:
        - path: /{proxy+}
          methods:
            - httpMethod: ANY
              functionname: $functionName
    sleepTimeMs: 10000

  test:
    action: rest/runner:send
    URL: ${setupAPI.EndpointURL}oginfo
    method: post
     '@request':
      region: ${awsSecrets.Region}
      URL: s3://mye2e-bucket/folder1/
    expect:
      Status: ok
      FileCount: 2
      LinesCount: 52

```

To see _Endly_ in action,
### End to end testing examples
 
In addition a few examples of fully functioning applications are included.
You can build, deploy and test them end to end all with endly.

 
1) **Web Service** 
   * [Reporter](example/ws/reporter) - a pivot table report builder.
        - Test with Rest Runner
        - Data Preparation and Validation (mysql)
        
2) **User Interface**
   * [SSO](example/ui/sso)  - user registration and login application.
        - Test with Selenium Runner
        - Test with HTTP Runner
        - Data Preparation and Validation (aersopike)
        - Web Content validation
        - Mocking 3rd party REST API with [http/endpoint service](testing/endpoint/http) 
        
3) **Extract, Transform and Load (ETL)**
   * [Transformer](example/etl/myApp) - datastore to datastore myApp (i.e. aerospike to mysql)
       - Test with Rest Runner
       - Data Preparation and Validation (aersopike, mysql)

4) **Runtime**  - simple http request event logger
   * [Logger](example/rt/elogger)
       - Test with HTTP Runner
       - Log Validation

5) **Serverless**  - serverless (lambda/cloud function/dataflow)
   * [Serverless](https://github.com/adrianwit/serverless_e2e)
    
    

<a name="Documentation"></a>

## Documentation
- [Installation](doc/installation)
- [Secret/Credential](doc/secrets)
- [Inline Workflow](doc/inline)
- [Workflow](doc/workflow)
- [Service](doc/service)
- [Usage](doc/usage)
- [User Defined Function](doc/udf)
- [Data store testing](testing/dsunit)



@run.yaml
```yaml
target:
  URL: "ssh://127.0.0.1/"
  credentials: localhost
systemPaths:
  - /usr/local/go/bin
commands:
  - go version
  - echo $GOPATH
```

## External resources

- [Endly introduction](https://github.com/adrianwit/endly-introduction)
- [Software Developement Automation - Part I](https://medium.com/@adrianwit/software-development-automation-e68b3bcc70c9?)
- [Software Developement Automation - Part II](https://medium.com/@adrianwit/software-development-automation-part-ii-bf961cfdd88a)
- [ETL end to end testing with docker, NoSQL, RDBMS and Big Query](https://medium.com/@adrianwit/etl-end-to-end-testing-with-docker-nosql-rdbms-and-big-query-35b13b7fada8)
- [Data testing strategy reinvented](https://medium.com/@adrianwit/killing-data-testing-swamp-6c3e11fb92c6)
- [Go lang e2e testing](https://github.com/adrianwit/golang-e2e-testing)
- [Endly UI e2e testing demo](https://www.youtube.com/watch?v=W6R4lk_iF0k&t=12s)

         	
## License

The source code is made available under the terms of the Apache License, Version 2, as stated in the file `LICENSE`.

Individual files may be made available under their own specific license,
all compatible with Apache License, Version 2. Please see individual files for details.


## TODO

- [ ] documentation improvements
- [ ] command executor with os/exec.Command
- [ ] gcp/containers integration
- [ ] gcp/cloudfunctions viant/afs integration
- [ ] ufd self describing meta data
- [ ] viant/afs docker connector


## Contributing to endly

endly is an open source project and contributors are welcome!


##  Credits and Acknowledgements

**Library Author:** Adrian Witas

