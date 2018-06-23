# Datastore services

- [Usage](#usage)
- [Datstore Credentials](#credentials)
- [Supported databases](#databases)


Datastore service uses [dsunit](https://github.com/viant/dsunit/) service to create, populate, and verify content of datastore. 


| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| dsunit | register | register database connection |  [RegisterRequest](https://github.com/viant/dsunit/blob/master/service_contract.go#L46) | [RegisterResponse](https://github.com/viant/dsunit/blob/master/service_contract.go#70)  |
| dsunit | recreate | recreate database/datastore |  [RecreateRequest](https://github.com/viant/dsunit/blob/master/service_contract.go#L76) | [RecreateResponse](https://github.com/viant/dsunit/blob/master/service_contract.go#98)  |    
| dsunit | sql | run SQL commands |  [RunSQLRequest](https://github.com/viant/dsunit/blob/master/service_contract.go#L103) | [RunSQLResponse](https://github.com/viant/dsunit/blob/master/service_contract.go#126)  |
| dsunit | script | run SQL script |  [RunScriptRequest](https://github.com/viant/dsunit/blob/master/service_contract.go#L132) | [RunSQLResponse](https://github.com/viant/dsunit/blob/master/service_contract.go#126)  |
| dsunit | mapping | register database table mapping (view), |  [MappingRequest](https://github.com/viant/dsunit/blob/master/service_contract.go#L155) | [MappingResponse](https://github.com/viant/dsunit/blob/master/service_contract.go#217)  |
| dsunit | init | initialize datastore (register, recreate, run sql, add mapping) |  [InitRequest](https://github.com/viant/dsunit/blob/master/service_contract.go#L225) | [MappingResponse](https://github.com/viant/dsunit/blob/master/service_contract.go#286)  |
| dsunit | prepare | populate databstore with provided data |  [PrepareRequest](https://github.com/viant/dsunit/blob/master/service_contract.go#L293) | [MappingResponse](https://github.com/viant/dsunit/blob/master/service_contract.go#323)  |
| dsunit | expect | verify databstore with provided data |  [ExpectRequest](https://github.com/viant/dsunit/blob/master/service_contract.go#L340) | [MappingResponse](https://github.com/viant/dsunit/blob/master/service_contract.go#380)  |
| dsunit | query | run SQL query |  [QueryRequest](https://github.com/viant/dsunit/blob/master/service_contract.go#L407) | [QueryResponse](https://github.com/viant/dsunit/blob/master/service_contract.go#419)  |
| dsunit | sequence | get sequence values for supplied tables |  [SequenceRequest](https://github.com/viant/dsunit/blob/master/service_contract.go#L388) | [SequenceResponse](https://github.com/viant/dsunit/blob/master/service_contract.go#400)  |


<a name="usage"></a>
## Usage

In order to operate on any data store the first step is to register named data store with specific driver:


```bash
endly -r=register
```


@register.yaml
```yaml
pipeline:
  db1:
    register:
      action: dsunit:register
      datastore: db1
      config:
        driverName: mysql
        descriptor: '[username]:[password]@tcp(127.0.0.1:3306)/[dbname]?parseTime=true'
        credentials: $mysqlCredentials
        parameters:
          dbname: db1

```


- **Create database schema and loading static data**

```bash
endly -r=init
```

@init.yaml

```bash
pipeline:
  create-db:
    db1:
      action: dsunit:init
      datastore: db1
      recreate: true
      config:
        driverName: mysql
        descriptor: '[username]:[password]@tcp(127.0.0.1:3306)/[dbname]?parseTime=true'
        credentials: mysql
      admin:
        datastore: mysql
        config:
          driverName: mysql
          descriptor: '[username]:[password]@tcp(127.0.0.1:3306)/[dbname]?parseTime=true'
          credentials: mysql
      scripts:
      - URL: datastore/db1/schema.sql
  prepare:
    db1:
      action: dsunit:prepare
      datastore: db1
      URL: datastore/db1/dictionary

```

In this scenario above workflow  
1) create or recreate mysql database db1, using mysql connection, 
2) execute schema DDL script to finally 
3) loads static data from datastore/db1/dictionary relative directory, where each file has to match a table name in db1 datastore, json and csv format are supported.

In this case dsunit.init also register datastore with driver thus no need to register it with separate workflow task.



<a name="loaddata">&nbsp;</a>
- **Loading data into data store**

Assuming that register or init task has already taken place within the same e2e workflow session


@prepare_db1.yaml
```bash
pipeline:
  prepare:
    db1:
      action: dsunit:prepare
      datastore: db1
      URL: datastore/db1/dictionary
      data: ${data.db1.setup}
```

- When **URL** attribute is used:  each file has to match a table name in db1 datastore and have json and csv extension.

- When **data** attribute is used, $data usually is defined by [workflow.Data](./../../model/workflow.go) attribute.


  _$data_ has to be in the following format:

  @setup_data.json
      
  ```json
        {
          "table1":[
              {"id":1, "name":"name 1"},
              {"id":2, "name":"name 2"}
          ],
          "table2":[
              {"id":1, "name":"name 1"},
              {"id":2, "name":"name 2"}
          ]
        }
  ``` 
    
     
    
  
  Print workflow model to examine "Data" attribute
    
  ```bash
  endly -r=regression -p  
  ```
  
  Run workflow
  ```bash
  endly -r=regression
  ```  
  
  Run workflow with debug option
    ```bash
    endly -r=regression -d=true
    ```

  @regression.csv

  | Workflow | Name |Tasks | | 
  | --- | ---| --- |--- | 
  |  | regression |%Tasks | |
  |**[]Tasks**| **Name** | **Actions** | |
  | | prepare | %Prepare | | 
  |**[]Prepare**| **Action** | **Request**  | **/Data.db1.[]setup** |   
  | | nop|  {} | @setup_data.json |
  | | run | @register.yaml |
  | | run | @prepare_db1.yaml |



_In many system where data is managed and refreshed by central cache service loading data per use case is very inefficient, to address this problem 
data defined on individual use case level can be loaded to database before individual use cases run if workflow.Data attribute is used._   

    
- **Dealing with data and autogenerated ID**    


In this scenario, workflow has to read initial datastore sequence to used it by  [_AsTableRecords_](udf.go) UDF.

    
@prepare_db1.yaml
```bash
    prepare:
      sequence:
        datastore: db1
        action: dsunit.sequence
        tables:
        - table1
        - table2
        post:
        - seq = $Sequences
      data:
        action: nop
        init:
        - db1key = data.db1.setup
        - db1Setup = $AsTableRecords($db1key)
      setup:
        datastore: db1
        action: dsunit:prepare
        data: $db1Setup
```


  _data.db1.setup_ has to use in the following [format](data.go):
    
    
  @setup_data.json    
```json
    [
      {
        "Table": "table1",
        "Value": {
          "id": "$id",
          "name": "Name 1",
          "type_id": 1
        },
        "AutoGenerate": {
          "id": "uuid.next"
        },
        "Key": "${tagId}_table1"
      },
      {
        "Table": "table2",
        "Value": {
          "id": "$seq.table2",
          "name": "Name 1",
          "type_id": 1
        },
        "PostIncrement": [
          "seq.table2"
        ],
        "Key": "${tagId}_table2"
      }
    ]
``` 


Using AsTableRecords is more advance testing option, allowing value autogeneration and data reference on individual use cases level by matching Key value: see the following example with unshift operator 

 
@test_init.json
```json
  [
      {
        "Name": "table1",
        "From": "<-dsunit.${tagId}_table1"
      },
      {
          "Name": "table2",
          "From": "<-dsunit.${tagId}_table2"
      }
  ]

```


<a name="mapping">&nbsp;</a>
- **Using data table mapping**

Dealing with large data model can be a huge testing bottleneck. 
Dsunit provide elegant way to address by defining [multi table mapping](https://github.com/viant/dsunit/blob/master/docs/README.md#mapping)

    
_Registering mapping_    

@mapping.yaml
```yaml
pipeline:
  mapping:
    datastore: db1
    action: dsunit.mapping
    mappings:
      - URL: regression/db1/mapping.json
```


<a name="validation">&nbsp;</a>
- **Validating data in data store**


Data validation can take place on various level

 - per use case:
 - after all use cases run with data pushed by individual use cases  
 


@epxec_db1.yaml
```bash
pipeline:
  prepare:
    db1:
      action: dsunit:epxect
      datastore: db1
      URL:  db1/expect
      data: ${data.db1.setup}
```
    
**URL** and **data** attribute works the same way as in data prepare. 


- See [assertly](https://github.com/viant/assertly#validation) and [dsunit](https://github.com/viant/dsunit) for comprehensive validation option
    - Directive and macro
    - Casting
    - Date/time formatting
    - testing nested unordered collection strucure


**Testing tables without primary key contrains**

1. **@fromQuery@** provides ability to defined in expected dataset SQL that returns both columns and rows in the same order as in expected dataset.

expected/user.json
```json
[
  {"@fromQuery@":"SELECT *  FROM users where id <= 2 ORDER BY id"},
  {"id":1, "username":"Dudi", "active":true, "salary":12400, "comments":"abc","last_access_time": "2016-03-01 03:10:00"},
  {"id":2, "username":"Rudi", "active":true, "salary":12600, "comments":"def","last_access_time": "2016-03-01 05:10:00"}
]
```

2. **@indexBy@**  provides ability to defined a unique key to index both actual and expected dataset right before validation
expected/user.json
```json
[
  {"@indexBy@":["id"]},
  {"id":1, "username":"Dudi", "active":true, "salary":12400, "comments":"abc","last_access_time": "2016-03-01 03:10:00"},
  {"id":2, "username":"Rudi", "active":true, "salary":12600, "comments":"def","last_access_time": "2016-03-01 05:10:00"}
]
```

<a name="credentials"></a>
## Datastore credentials

Credential are stored in ~/.secret/CREDENTIAL_NAME.json using [toolobx/cred/config.go](https://github.com/viant/toolbox/blob/master/cred/config.go) format.

For example:

@source_mysql
```json
{"Username":"root","Password":"dev"}
 ```

To generate encrypted credentials download and install the latest [endly](https://github.com/viant/endly/releases) and run the following

```bash
endly -c=mysql
```

For BigQuery: use service account generated JSON credentials  


<a name="databases"></a>
## Supported databases

- any database that provide database/sql golang driver.


Already included drivers with [endly](./../../bootstrap/bootstrap.go) default build.

 - mysql
 - postgresql
 - aerospike
 - bigquery
 - mongo


Tested, but not included drivers with custom endly build:

 - vertica
 - oracle

 