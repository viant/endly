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
| CsvReader | `headerFields, delimiter` or `{output: rows, delimiter: ','}` |

Register `CsvReader` with `output: rows` to verify positional CSV records.

```yaml
pipeline:
  register:
    action: udf:register
    udfs:
      - id: ReadCSVRows
        provider: CsvReader
        params:
          - output: rows
  read:
    action: storage:download
    source: {URL: 'fixtures/export.csv'}
    udf: ReadCSVRows
    destKey: observedRows
```

The result contains every parsed record in order, including the observed header
as the first record. Cells remain strings and each record retains its original
width, including trailing empty fields. Blank lines remain zero-width records.
An initial UTF-8 BOM is ignored. The
delimiter defaults to a comma. Malformed CSV or invalid UTF-8 fails the action.
The positional header arguments still produce the existing log reader output.

`$ShellQuote('value')` reads a string from a literal state key or path and
produces one literal POSIX shell argument for `exec:run`. Use it for values such
as paths that may contain spaces, quotes or shell syntax.
The surrounding command and environment keep their normal expansion behavior.

```yaml
commands:
  - "printf '%s' $ShellQuote('value')"
```


### Service actions

**UDF Service**

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| udf | register | register custom UDF with udf provider | [RegisterRequest](service_contract.go) | [RegisterResponse](service_contract.go) |

### Literal state JSON bytes

`$StateJSONBytes('result')` resolves one literal state key/path through the
native state lookup, serializes the stored value, and returns `[]byte`. Dotted
paths such as `'result.StructuredContent'` select a stored subtree. Pass the
key/path string, not `$result`: ordinary UDF arguments may expand referenced
map fields before the UDF sees them.

```yaml
pipeline:
  retainResult:
    action: storage:upload
    init:
      - name: retainedBytes
        value: $StateJSONBytes('result')
        required: true
        emptyIfUnexpanded: true
    sourceKey: retainedBytes
    dest:
      URL: ${outputDirectory}/result.json
```

The JSON uses the same tab-indented formatting as `AsJSON`, preserving literal
code/templates, UTF-8, numeric values, zero, false, null and empty arrays/objects.
Stored values are not expanded or mutated. Empty, missing, non-string or
expression-like keys and unsupported JSON values fail with payload-free errors.
Keep required/unexpanded validation when retaining a receipt: a failed UDF must
not be saved as an unevaluated expression.

Binary upload content remains literal. String uploads retain their existing
workflow-template expansion; `AsJSON` and authored variable/fixture interpolation
are unchanged. Computed results should first be staged under a state key and
then serialized with `StateJSONBytes`.

For literal JSON comparisons, stage the bytes before applying existing UDFs;
embedded nested UDF calls are not a literal-data boundary:

```yaml
init:
  - sourceBytes = $StateJSONBytes('response.definition')
  - sourceFingerprint = $Base64Encode($sourceBytes)
```

`Base64Encode` receives bytes and preserves the exact JSON encoding. Staged
`$AsString($sourceBytes)` also preserves JSON text; the native validator may
compare bare JSON strings structurally. For exact text comparison, use
`actual: definition:${actualJSON}` and `expect: definition:${expectedJSON}`
after staging each text with `AsString`, or compare the base64 form. Do not pass serialized code back through a string-template upload.

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
