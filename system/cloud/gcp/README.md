# Google Cloud services

The set of google cloud services provice procy of various google.golang.org/api APIs 

#### Global variables

- $gcp.projectID: expands with project ID
 

#### Usage:



To check all supported method run
```bash
     endly -s='gcp/GOOGLE COULD SERVICE'
```

i.e 

To check method contract run endly -s="gc/compute" -a=methodName
```bash
    endly -s="gc/compute" -a='instancesGet'
```

```bash
endly -r=test
```


@test.yaml
```yaml
pipeline:
  start:
    info:
      action: gcp/compute:instancesGet
      logging: false
      credentials: gcp
      zone: us-central1-f
      instance: $instanceId
      project: myProject
      scopes:
        - https://www.googleapis.com/auth/compute
        - https://www.googleapis.com/auth/devstorage.full_control
      urlParams:
        filter: project:* 
```

The first action for given service has to define service account credentials i.e (~/.secret/gcp.json)
Project and scopes are set by default from secrets file, so they can be skipped
