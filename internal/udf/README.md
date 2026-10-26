#User Defined Function Service

This service enables custom UDF registration with pre defined provider set.

- [Usage](#usage)
- [Predefined UDF Providers](#providers)
- [Service Actions](#actions)


### Usage

Register UDF

```go
endly -r=reqister
```

@register.yaml
```yaml
pipeline:
    register-udf:
      action: udf:register
      udfs:
        - id: AddressBookToProto
          provider: ProtoWriter
          params:
            - /Project/proto/address_book.proto
            - AddressBook
        - id: ProtoToAddressBook
          provider: ProtoReader
          params:
            - /Project/proto/address_book.proto
            - AddressBook
```


Avro Reader UDF data validation:


```endly -r=test```

@test.yaml
```yaml
pipeline:
  loadData:
    action: storage:download
    udf: AvroReader
    source:
      URL: gs://mye2ebucket/data/output/1/app_data00000.avro
    credentials: $gcpSecrets
    destKey: matchedData
    post:
      myData: $Transformed

  infoMyData:
    action: print
    message: $AsJSON(${myData})

  infoMatched:
    action: print
    message: $AsJSON(${matchedData})

  infoAll:
    action: print
    message: $AsJSON(${loadData})
    
  assert:
    action: validator:assert
    actual: $AsData(${matchedData})
    expect: $Cat(${parent.path}/expect/mydata.json)
    
```

### Predefined UDF Providers

| Provider | Arguments | 
|---|---| 
| ProtoReader |  schemaFile, messageType, importPath |
| ProtoReader | schemaFile, messageType, importPath |
| AvroWriter | avroSchema/URL, compression |
| CsvReader | headerFields, delimiter |


### Service actions

**UDF Service**

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| udf | register | register custom UDF with udf provider | [RegisterRequest](service_contract.go) | [RegisterResponse](service_contract.go) |

### JSON Lines captures

`$LoadJSONLines('capture.jsonl')` always returns a collection, including zero or
one record. Use it for transaction captures before `Select`, `Len`, or a
`validator:assert` with `each: true`. `LoadJSON` keeps its existing regular-JSON
shape behavior; a single object there remains a map.

```yaml
pipeline:
  inspect:
    action: validator:assert
    init:
      - name: traffic
        value: $LoadJSONLines('${captureFile}')
      - name: reads
        value: $Select('traffic[method = ReadRecord]')
      - name: readCount
        value: $Len($reads)
    actual: $readCount
    expect: 1
```

Records must be JSON objects; malformed input or a non-object record rejects the
whole load. Blank input returns an empty collection, and an empty object remains
a record so missing required fields cannot be silently hidden. Local paths and
owner-relative resources use the normal file resolution. Large records are not
limited by a line scanner's default buffer size.
