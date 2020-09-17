# Google Cloud Run Service

This service is google.golang.org/api/run/v1/APIService proxy 

To check all supported method run
```bash
     endly -s='gcp/run'
```

To check method contract run endly -s='gcp/run' -a=methodName
```bash
    endly -s='gcp/run:deploy' 
```

_References:_
- [Clud Run API](https://cloud.google.com/run/docs/reference/rest/)


#### Usage:


##### Deploying GCR image 

```bash
endy -r=deploy
```

[@deploy.yaml](deploy.yaml)
```yaml
pipeline:
    deploy:
      action: gcp/run:deploy
      credentials: gcp-e2e
      image: gcr.io/cloudrun/hello
      memoryMb: 256M
      replace: true   #if service already exist, redeploy
      public: true
    info:
      action: print
      message: $deploy.Endpoint
```


TODO add more examples
##### Deploying GCR image on GKE



##### Build and deploying GCR image 


