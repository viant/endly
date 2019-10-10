# Google Storage Service 

This service is google.golang.org/api/pubsub/v1.Service proxy 

To check all supported method run
```bash
     endly -s='gcp/storage'
```

To check method contract run endly -s='gcp/storage:methodName'
```bash
    endly  -s='gcp/storage:bucketAccessControlsList'

```

_References:_
- [Pub/Storage API](https://cloud.google.com/storage/docs/reference/rest/)


## Usage

1. Setting bucket notification

```endly notification authWith=myGCPSecrets```

where
[@notification.yaml](usage/notification.yaml)

```yaml

```