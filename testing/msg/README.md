# Msg - Messaging Service



### Google Cloud pubsub

The following workflow define simple topic/subscription producing and consuming example.

Example credentials 'gcp-e2e' is name of [google secrets](./../../doc/secrets) placed to  ~/.secret/gcp-e2e.json


```endly pubsub```


[@pubsub.yaml](usage/gcp/pubsub.yaml)
```yaml
pipeline:
  create:
    action: msg:setupResource
    resources:
      - URL: myTopic
        type: topic
        vendor: gc
        credentials: gcp-e2e

      - URL: mySubscription
        type: subscription
        vendor: gc
        credentials: gcp-e2e
        config:
          topic:
            URL: /projects/${msg.projectID}/topics/myTopic

  setup:
    action: msg:push
    dest:
      URL: /projects/${msg.projectID}/topics/myTopic
      credentials: gcp-e2e
    messages:
      - data: "this is my 1st message"
        attributes:
          attr1: abc
      - data: "this is my 2nd message"
        attributes:
          attr1: xyz

  validate:
    action: msg:pull
    count: 2
    nack: true
    source:
      URL: /projects/${msg.projectID}/subscriptions/mySubscription
      credentials: gcp-e2e
    expect:
      - '@indexBy@': 'Attributes.attr1'
      - Data: "this is my 1st message"
        Attributes:
          attr1: abc
      - Data: "this is my 2nd message"
        Attributes:
          attr1: xyz
```


### Amazon Simple Queue Service


The following workflow define simple topic/subscription producing and consuming example.

Example credentials 'aws-e2e' is name of [aws secrets](./../../doc/secrets) placed to  ~/.secret/aws-e2e.json


```bash
endly queue
```


[@queue.yaml](usage/aws/queue.yaml)
```yaml
init:
  awsCredentials: aws-e2e
pipeline:
  setup:
    action: msg:setupResource
    credentials: $awsCredentials
    resources:
      - URL: mye2eQueue1
        type: queue
        vendor: aws

  trigger:
    action: msg:push
    credentials: $awsCredentials
    sleepTimeMs: 5000
    dest:
      URL: mye2eQueue1
      type: queue
      vendor: aws
    messages:
      - data: 'Test: this is my 1st message'
      - data: 'Test: this is my 2nd message'

  validate:
    action: msg:pull
    credentials: $awsCredentials
    timeoutMs: 20000
    count: 2
    source:
      URL: mye2eQueue1
      type: queue
      vendor: aws
    expect:
      - '@indexBy@': 'Data'
      - Data: "Test: this is my 1st message"
      - Data: "Test: this is my 2nd message"
  info:
    action: print
    message: $AsJSON($validate)
```


### Amazon Simple Notification Service

