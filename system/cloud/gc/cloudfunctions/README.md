# Google Cloud Functions Service 

This service is google.golang.org/api/cloudfunctions/v1beta2.Service proxy 

To check all supported method run
```bash
     endly -s='gc/cloudfunctions'
```

To check method contract run endly -s='gc/cloudfunctions' -a=methodName
```bash
    endly -s='gc/cloudfunctions' -a='operationsList'

```

_References:_
- [Cloud Functions API](https://cloud.google.com/functions/docs/reference/rest/)


#### Example


###### Calling function


```bash
endly -r=call
```

@call.yaml
```yaml
defaults:
  credentials: am
pipeline:
  getInfo:
    action: gc/cloudfunctions:functionsGet
    '@name': projects/myProject/locations/us-central1/functions/HelloWorld
  test:
    action: print
    message: 'TriggerURL: $info.HttpsTrigger.Url'
  callFunction:
    action: gc/cloudfunctions:functionsCall
    '@name': 'projects/myProject/locations/us-central1/functions/HelloWorld1'
    callfunctionrequest:
      data:
  printOutput:
    action: print
    message: $callFunction.Result
```


###### Listing functions

```bash
endy -r=listFunctions
```

@listFunctions.yaml
```yaml
defaults:
  credentials: am
pipeline:
  list:
    info:
      action: gc/cloudfunctions:functionsList
      parent: projects/myProject/locations/-
```


###### Operation list:

```bash
endy -r=list
```

@list.yaml
```yaml
defaults:
  credentials: am
pipeline:
  list:
    info:
      action: gc/cloudfunctions:operationsList
      urlParams:
        filter: project:myProject,latest:true
```


