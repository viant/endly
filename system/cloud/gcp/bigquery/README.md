# Google BigQuery Service 

This service is google.golang.org/api/bigquery/v2.Service proxy 

To check all supported method run
```bash
     endly -s='gc/bigquery'
```

To check method contract run endly -s='gc/bigquery' -a=methodName
```bash
    endly -s='gc/bigquery' -a='datasetsList'

```

_References:_
- [BigQuery API](https://cloud.google.com/bigquery/docs/reference/rest/v2/)


#### Usage:

```bash
endy -r=list
```

@list.yaml
```yaml
pipeline:
  start:
    info:
      action: gc/bigquery:datasetsList
      credentials: gc
      projectID: myProject
```

