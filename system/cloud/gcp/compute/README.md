# Google Cloud Compute Engine Service 

This service is google.golang.org/api/compute/v1/compute.Service proxy 

To check all supported method run
```bash
     endly -s='gc/compute'
```

To check method contract run endly -s="gc/compute" -a=methodName
```bash
    endly -s="gc/compute" -a='instancesGet'
```


_References:_
- [Compute Engine API](https://cloud.google.com/compute/docs/reference/rest/v1/)


#### Usage:


###### Starting instance
```bash
endly -r=start
```

@start.yaml
```yaml
init:
  instanceId: 11230632249892XXXXX
pipeline:
  start:
    info:
      action: gc/compute:instancesGet
      logging: false
      credentials: gc
      zone: us-central1-f
      instance: $instanceId

    print:
      action: print
      message: Instance $instanceId is  $info.Status

    check:
      when: $info.Status = 'RUNNING'
      action: exit

    instanceUp:
      when: $info.Status = 'TERMINATED'
      action: gc/compute:instancesStart
      logging: false
      zone: us-central1-f
      instance: $instanceId

    waitForStart:
      action: nop
      logging: false
      sleepTimeMs: 5000

    gotoStart:
      action: goto
      task: start
```


###### Stoping instance
```yaml
init:
  instanceId: 11230632249892XXXXX
stop:
    info:
      action: gc/compute:instancesGet
      logging: false
      credentials: gc
      zone: us-central1-f
      instance: $instanceId

    print:
      action: print
      message: Instance $instanceId is $info.Status

    check:
      when: $info.Status = 'TERMINATED'
      action: exit

    instanceDown:
      when: $info.Status = 'RUNNING'
      action:  gc/compute:instancesStop
      credentials: gc
      zone: us-central1-f
      instance: $instanceId
        
    waitForStop:
      action: nop
      logging: false
      sleepTimeMs: 5000

    gotoStop:
      action: goto
      task: stop
```
