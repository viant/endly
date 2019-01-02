# Msg - Messaging Service

work in progress ...


### Google Cloud pubsub

The following workflow define simple topic/subscription producing and consuming example.

Example credentials 'am' is name of [google secrets](./../../doc/secrets) placed to  ~/.secret/am.json


```endly -r=pubsub.yaml```


[@pubsub.yaml](pubsub.yaml)
```yaml
pipeline:
  create:
    action: msg:create
    resources:
      - URL: myTopic
        type: topic
        vendor: gc
        credentials: am

      - URL: mySubscription
        type: subscription
        vendor: gc
        credentials: am
        config:
          topic:
            URL: /projects/${msg.projectID}/topics/myTopic

  setup:
    action: msg:push
    dest:
      URL: /projects/${msg.projectID}/topics/myTopic
      credentials: am
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
      credentials: am
    expect:
      - '@indexBy@': 'Attributes.attr1'
      - Data: "this is my 1st message"
        Attributes:
          attr1: abc
      - Data: "this is my 2nd message"
        Attributes:
          attr1: xyz
```
