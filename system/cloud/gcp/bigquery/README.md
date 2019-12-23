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

#####  Listing dataset

```endly -run='gcp/bigquery:datasetsList' projectID=myProject```

  
##### Query with destination table

``` endly -r=query authWith=myGoogleSecrets.json```


[@query.yaml](usage/query.yaml)
```yaml
init:
  dataset: myDataset
  '!gcpCredentials': $params.authWith

pipeline:
  query:
    action: gcp/bigquery:query
    credentials: $gcpCredentials
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

##### Creating Materialize View
``` endly -r=mv authWith=myGoogleSecrets.json```


[@mv.yaml](usage/mv.yaml)
```yaml
init:
  '!gcpCredentials': $params.authWith

pipeline:
  query:
    action: gcp/bigquery:tablesInsert
    credentials:  gcpCredettials

    kind: bigquery#table
    projectId: ${gcp.projectID}
    datasetId: myDataset
    table:
      tableReference:
        projectId: ${gcp.projectID}
        datasetId: myDataset
        tableId: myTable_mv
      materializedView:
        query: SELECT SUM(columnA) AS columnA, MIN(col
```

##### Table copy

``` endly -r=copy authWith=myGoogleSecrets.json```

[@copy.yaml](usage/copy.yaml)
```yaml
init:
  '!gcpCredentials': $params.authWith
  gcpSecrets: ${secrets.$gcpCredentials}
  i: 0

  src:
    projectID: $gcpSecrets.ProjectID
    datasetID: db1
  dest:
    projectID: $gcpSecrets.ProjectID
    datasetID: db1e2e

pipeline:
  registerSource:
    action: dsunit:register
    datastore: ${src.datasetID}
    config:
      driverName: bigquery
      credentials: $gcpCredentials
      parameters:
        datasetId: $src.datasetID

  readTables:
    action: dsunit:query
    datastore: ${src.datasetID}
    SQL: SELECT table_id AS table FROM `${src.projectID}.${src.datasetID}.__TABLES__`
    post:
      dataset: $Records


  copyTables:
    loop:
      action: print
      message: $i/$Len($dataset) -> $dataset[$i].table

    copyTable:
      action: gcp/bigquery:copy
      logging: false
      credentials: $gcpCredentials
      sourceTable:
        projectID: ${src.projectID}
        datasetID: ${src.datasetID}
        tableID: $dataset[$i].table
      destinationTable:
        projectID: ${dest.projectID}
        datasetID: ${dest.datasetID}
        tableID: $dataset[$i].table

    inc:
      action: nop
      init:
        _ : $i++
    goto:
      when: $i < $Len($dataset)
      action: goto
      task: copyTables

```

##### Schema patch


1. Patched with table template:


``` endly -r=patch_with_template authWith=myGoogleSecrets.json```

[@patch_with_template.yaml](usage/patch_with_template.yaml)
```yaml
pipeline:
  patch:
    action: gcp/bigquery:patch
    credentials: $gcpCredentials
    table: ${projectID}:bqtail.dummy_temp
    template: ${projectID}:bqtail.dummy

```

2. Patched with table schema:

``` endly -r=patch_with_schema authWith=myGoogleSecrets.json```

[@patch_with_schema.yaml](usage/patch_with_schema.yaml)
```yaml
pipeline:

  getTable:
    action: gcp/bigquery:table
    credentials: $gcpCredentials
    table: ${projectID}:bqtail.dummy
    dest:
      URL: /tmp/mytable.json


  patch:
    init:
      table: $Cat('/tmp/mytable.json')
      tableMap: $AsMap($table)
      schema: $tableMap.schema
    action: gcp/bigquery:patch
    credentials: $gcpCredentials
    table: ${projectID}:bqtail.dummy_temp
    schema: $schema
```



3. Patch in loop;


``` endly -r=patch authWith=myGoogleSecret template='my_project:my_dataset.myTableTempalte'  dataset=mydataset criteria='narrowing_value'  '```


[@patch.yaml](usage/patch.yaml)

```yaml
init:
  'gcpCredentials': $params.authWith
  gcpSecrets: ${secrets.$gcpCredentials}
  projectID: $gcpSecrets.ProjectID
  '!template': $params.template
  '!dataset': $params.dataset
  '!criteria': $params.criteria
  i: 0


pipeline:
  registerSource:
    action: dsunit:register
    datastore: ${dataset}
    config:
      driverName: bigquery
      credentials: $gcpCredentials
      parameters:
        datasetId: $dataset

  readTables:
    action: dsunit:query
    datastore: $dataset
    SQL: SELECT project_id, dataset_id, table_id  FROM `${projectID}.${dataset}.__TABLES__` WHERE table_id LIKE '%${criteria}%'
    post:
      tables: $Records


  patchTable:
    loop:
      action: print
      message: Patching $i/$Len($tables) -> $tables[$i].table_id

    patch:
      action: gcp/bigquery:patch
      logging: false
      template: $template
      tableId: $tables[$i].table_id
      datasetId: $tables[$i].dataset_id
      projectId: $tables[$i].project_id
      credentials: $gcpCredentials

    inc:
      action: nop
      init:
        _: $i++
    goto:
      when: $i < $Len($tables)
      action: goto
      task: patchTable

```