# Google Kubernetes Engine Service

This service is google.golang.org/api/cloudscheduler/v1/Service proxy 

To check all supported method run
```bash
     endly -s='gcp/cloudscheduler'
```

To check method contract run endly -s='gcp/cloudscheduler' -a=methodName
```bash
    endly -s='gcp/cloudscheduler:clustersGet' 
```

_References:_
- [Google Kubernetes Engine API](https://cloud.google.com/kubernetes-engine/docs/reference/rest/)


#### Usage:

##### Deploying schedule job

```endly deploy authWith=myGoogleSecret.json```

[@deploy.yaml](usage/deploy.yaml)
```yaml
pipeline:
  deploy:
    action: gcp/cloudscheduler:deploy
    credentials: viant-e2e
    name: Replay
    schedule: 0 * * * *
    body: body comes here
    httpTarget:
      headers:
      "User-Agent": Google-Cloud-Scheduler"
      httpMethod: POST
      uri: https://us-central1-viant-e2e.cloudfunctions.net/BqTailReplay

```