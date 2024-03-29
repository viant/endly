# Amazon Elastic Compute service

This service is github.com/aws/aws-sdk-go/service/ec2.EC2 proxy 

To check all supported method run
```bash
    endly -s="aws/ec2"
```

To check method contract run endly -s="aws/ec2" -a=methodName
```bash
    endly -s="aws/ec2" -a='stopInstances'
```

#### Usage:

##### Starting instance

@start.yaml
```yaml
init:
  instanceId: i-0064b6c35XXXXX
pipeline:
  start:
    info:
      action: aws/ec2:describeInstances
      logging: false
      credentials: aws
      instanceids:
       - $instanceId

    print:
      action: print
      message: Instance $instanceId is  $info.Reservations[0].Instances[0].State.Name

    check:
      when: $info.Reservations[0].Instances[0].State.Name = 'running'
      action: exit

    instanceUp:
      when: $info.Reservations[0].Instances[0].State.Name = 'stopped'
      action: aws/ec2:startInstances
      logging: false
      instanceids:
        - $instanceId

    waitForStart:
      action: nop
      logging: false
      sleepTimeMs: 5000

    gotoStart:
      action: goto
      task: start

```


##### Stop instance
@stop.yaml
```yaml
init:
  instanceId: i-0064b6c35XXXXX
pipeline:
  stop:
    info:
      action: aws/ec2:describeInstances
      logging: false
      credentials: aws
      instanceids:
        - $instanceId

    print:
      action: print
      message: Instance $instanceId is  $info.Reservations[0].Instances[0].State.Name

    check:
      when: $info.Reservations[0].Instances[0].State.Name = 'stopped'
      action: exit

    instanceDown:
      when: $info.Reservations[0].Instances[0].State.Name = 'running'
      action: aws/ec2:stopInstances
      logging: false
      instanceids:
        - $instanceId
    waitForStop:
      action: nop
      logging: false
      sleepTimeMs: 5000
    gotoStop:
      action: goto
      task: stop
```

### Getting instance by tag name


```endly -r=instance```

[@instance.yaml](instance.yaml)
```yaml
pipeline:
  instanceInfo:
    action: aws/ec2:getInstance
    credentials: aws-e2e
    '@name': e2e-aero
  info:
    action: print
    message: $AsJSON($instanceInfo)
```

### Getting vpc by tag name

```yaml
pipeline:
  vpcInfo:
    action: aws/ec2:getVpc
    credentials: aws-e2e
    '@name': aero
  info:
    action: print
    message: $AsJSON($vpcInfo)
```

### Getting vpcConifg


```endly -r=vpc_config```

[@vpc_config](vpc_config.yaml)
```yaml
pipeline:
  byVpc:
    vpcConfigInfo1:
      action: aws/ec2:getVpcConfig
      credentials: aws-e2e
      vpc:
        name: aero
    info:
      action: print
      message: $AsJSON($vpcConfigInfo1)

  byInstance:
    vpcConfigInfo2:
      action: aws/ec2:getVpcConfig
      credentials: aws-e2e
      instance:
        name: e2e-aero
    info:
      action: print
      message: $AsJSON($vpcConfigInfo2)

```