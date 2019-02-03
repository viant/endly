# Google BigQuery Service 

This service is google.golang.org/api/bigquery/v2.Service proxy 

To check all supported method run
```bash
     endly -s='gcp/bigquery'
```

To check method contract run endly -s='gcp/bigquery' -a=methodName
```bash
    endly -s='gcp/bigquery' -a='datasetsList'

```

_References:_
- [BigQuery API](https://cloud.google.com/bigquery/docs/reference/rest/v2/)


#### Usage:

1. Listing dataset

```bash
endy -run='gcp/bigquery:datasetsList' projectID=myProject
```


2. Query with destination table

```bash
endy -r=query
```

[@query.yaml](query.yaml)
```yaml
init:
  dataset: myDataset
defaults:
  credentials: gc
pipeline:
  query:
    action: gcp/bigquery:query
    query: SELECT * FROM mySourceTable
    allowlargeresults: false
    defaultdataset:
      projectid: ${gcp.projectID}
      datasetid: $dataset
    destinationtable:
      projectid: ${gcp.projectID}
      datasetid: $dataset
      tableid: myTable
    writedisposition: WRITE_APPEND
```
