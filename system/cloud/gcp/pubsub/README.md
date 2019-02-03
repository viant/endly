# Google Cloud Pub/Sub Service 

This service is google.golang.org/api/pubsub/v1.Service proxy 

To check all supported method run
```bash
     endly -s='gcp/pubsub'
```

To check method contract run endly -s='gcp/pubsub' -a=methodName
```bash
    endly -s='gcp/pubsub' -a='subscriptionsList'

```

_References:_
- [Pub/Sub API](https://cloud.google.com/pubsub/docs/reference/rest/)


#### Usage:

```bash
endy -r=list
```

@list.yaml
```yaml
pipeline:
  start:
    info:
      action: gcp/pubsub:topicsList
      credentials: gc
      projectID: myProject-p1
```

