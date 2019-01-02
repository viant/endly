## Test project generator 

Test project generator has ability to create initial test project with various setup, data and testing options.

### Usage

#### Generate test project


```bash
endly -g
```

or via [online test generator](https://endly-external.appspot.com/)

![](../../project_generator.png)

 1. In Template dropdown select external application URL template
 2. In Origin type your application URL i.e. file:///Project/myapp or http://github.com/repo/myapp
    - if application codebase defines Dockerfile and docker-compose.yaml endly includes them in the app.yaml workflow. 
 3. Select SDK
 4. Select application databases
 5. Select testing options 

A initial test project workflows:

![](initial_project.png)

  -  run.yaml is an entry point orchestrating other workflows.
  -  system.yaml deploys application services
  -  datastore.yaml creates databases schema and loads static data.
  -  app.yaml builds and deploy application
  -  regression/regression.csv sets use cases initial state and runs tests.
  -  data.yaml registers data stores and sets initial data state  


**Regression workflow:**

  regression.yaml|csv define a regression test workflow using either [inline](../inline) or [neatly](https://github.com/viant/neatly) format.

The typical workflow performs the following task:

1.  Register data store driver, set database IP by inspecting corresponding docker container, and optionally sets initial data state: data.yaml

2. Check if skip.txt file exist to skip specific use case

3. Optionally set initial test data for all data stores if regression/use_cases/xxx/prepare/$db is defined with corresponding tables data.

4. Run a REST/HTTP/Selenium test

5. Verify data in data stores only if expected data is defined in regression/use_cases/xxx/expect/$db



#### Running all tasks

```bash
endly 
```

#### System, data store initialization and application build and deployment.

```bash
endly -t=init
```

#### Running only all tests

```bash
endly -t=test
```

#### Running individual test

endly -t=test -i=&lt;tagID>

#### Running individual test with debug logs

endly -t=test -d=true -i=&lt;tagID>  
