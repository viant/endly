# Msg - Messaging Service



### Google Cloud pubsub

The following workflow define simple topic/subscription producing and consuming example.

Example credentials 'gcp-e2e' is name of [google secrets](./../../doc/secrets) placed to  ~/.secret/gcp-e2e.json


```endly test```


[@test.yaml](usage/external/test.yaml)

```yaml
init:
  gcpCredentials: gcp-e2e
pipeline:
  create:
    action: msg:setupResource
    resources:
      - URL: myTopic
        type: topic
        vendor: gcp
        credentials: $gcpCredentials
      - URL: mySubscription
        type: subscription
        vendor: gcp
        credentials: $gcpCredentials
        config:
          topic:
            URL: /projects/${msg.projectID}/topics/myTopic

  setup:
    action: msg:push
    dest:
      URL: /projects/${msg.projectID}/topics/myTopic
      credentials: $gcpCredentials
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
      credentials: $gcpCredentials
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


The following workflow creates a queue to produce and consume test messages.

Example credentials 'aws-e2e' is name of [aws secrets](./../../doc/secrets) placed to  ~/.secret/aws-e2e.json


```bash
endly queue.yaml
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

The following workflow creates a topic to produce test messages.

Example credentials 'aws-e2e' is name of [aws secrets](./../../doc/secrets) placed to  ~/.secret/aws-e2e.json


```bash
endly topic.yaml
```

[@topic.yaml](usage/aws/topic.yaml)
```yaml
init:
  awsCredentials: aws-e2e
pipeline:
  setup:
    action: msg:setupResource
    credentials: $awsCredentials
    resources:
      - URL: mye2eTopic
          type: topic
          vendor: aws

  trigger:
    action: msg:push
    credentials: $awsCredentials
    sleepTimeMs: 5000
    dest:
      URL: mye2eTopic
      type: topic
      vendor: aws
    messages:
      - data: 'Test: this is my 1st message'
        attributes:
          id: abc1
      - data: 'Test: this is my 2nd message'
        attributes:
          id: abc2
```


## Kafka

The following workflow define kafka topic and produces and consume messages.

```bash
endly test
```


[@test.yaml](usage/kafka/test.yaml)
```yaml
pipeline:
  startUp:
    action: docker/ssh:composeUp
    comments: setup kafka cluster
    sleepTimeMs: 10000
    runInBackground: true
    source:
      URL: docker-compose.yml

  create:
    sleepTimeMs: 10000
    action: msg:setupResource
    comments: create topic and wait for a leadership election
    resources:
      - URL: myTopic
        type: topic
        replicationFactor: 1
        partitions: 1
        brokers:
          - localhost:9092


  setup:
    action: msg:push
    dest:
      url: tcp://localhost:9092/myTopic
      vendor: kafka

    messages:
      - data: "this is my 1st message"
        attributes:
          key: abc
      - data: "this is my 2nd message"
        attributes:
          key: xyz

  validate:
    action: msg:pull
    count: 2
    nack: true
    source:
      url: tcp://localhost:9092/myTopic
      vendor: kafka
    expect:
      - '@indexBy@': 'Attributes.key'
      - Data: "this is my 1st message"
        Attributes:
          key: abc
      - Data: "this is my 2nd message"
        Attributes:
          key: xyz

  cleanUp:
    action: docker/ssh:composeDown
    source:
      URL: docker-compose.yml
```

