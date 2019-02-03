# Google Cloud Functions Service 

This service is google.golang.org/api/cloudfunctions/v1beta2.Service proxy 

To check all supported method run
```bash
     endly -s='gcp/cloudfunctions'
```

To check method contract run endly -s='gcp/cloudfunctions' -a=methodName
```bash
    endly -s='gcp/cloudfunctions' -a='operationsList'
```


_References:_
- [Cloud Functions API](https://cloud.google.com/functions/docs/reference/rest/)


#### Example

###### Testing

```go
endly -r=test

```
[@test.yaml](test.yaml)
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

###### Deploying function


1. Deploying http trigger function with archive
    ```bash
    endly -r=deploy
    ```
    [@deploy.yaml](deploy.yaml)
    ```yaml
    pipeline:
      deploy:
        action: gcp/cloudfunctions:deploy
        '@name': HelloWorld
        runtime: go111
        source:
          URL: test/hello.zip
    
    ```
2. Deploying http trigger function with source path (use .gcloudignore to control upload)
    @deploy.yaml
    ```yaml
    pipeline:
      deploy:
        action: gcp/cloudfunctions:deploy
        '@name': HelloWorldFn
        entryPoint Hello
        runtime: go111
        source:
          URL: test/
    ```
3. Deploying with eventTrigger
    @deploy_with_trigger
    ```yaml
    pipeline:
      deploy:
        action: gcp/cloudfunctions:deploy
        '@name': MyFunction
        entryPoint: MyFunctionFN
        runtime: go111
        eventTrigger:
          eventType: google.storage.object.finalize
          resource: projects/_/buckets/myBucket
        source:
          URL: test/
    
    ```


###### Calling function

1. Calling from workflow
    ```bash
    endly -r=call
    ```
    @call.yaml
    ```yaml
    pipeline:
      call:
        action: gcp/cloudfunctions:call
        logging: false
       '@name': HelloWorld
        data:
          from: Endly
    ```
2. Calling from cli
    ```bash
    endly -run='gcp/cloudfunctions:call' name=HelloWorld data.from=Endly
    ``` 


###### Getting function info

```bash
    endly -run='gcp/cloudfunctions:get' name=HelloWorld 
```


###### Listing functions

```bash
    endly -run='gcp/cloudfunctions:list'  
```
