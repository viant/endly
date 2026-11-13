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
| XlsxReader | `{sheet: Worksheet name, range: autoFilter or A1 range}` or `{ranges: [{id, sheet, range}], metadata: true}` |

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

Register `XlsxReader` to read a static grid from a named worksheet in an XLSX
download:

```yaml
pipeline:
  register:
    action: udf:register
    udfs:
      - id: ReadInventoryGrid
        provider: XlsxReader
        params: [{sheet: Inventory, range: autoFilter}]
  read:
    action: storage:download
    source: {URL: 'fixtures/inventory.xlsx'}
    udf: ReadInventoryGrid
    destKey: observedGrid
```

The result contains `sheet`, the observed worksheet AutoFilter `range`, the
original `byteCount`, and rectangular `rows` and `types` grids, including the
first row and all trailing blank positions inside the range. Cell types are
`blank`, `string`, `number`, `boolean`, and `date`. A blank has a null value;
an explicit empty string has a string value. Numbers use their raw numeric
value, including numbers displayed with a date or percentage format. Native
OOXML date cells retain their date text. Validate the observed range, cell
types and values against independent expectations.

Use an explicit A1 range such as `A4:A4` for a one-cell grid. With
`range: autoFilter`, the worksheet must declare its AutoFilter. The reader
uses the complete range without applying the filter.

Read several ranges and workbook metadata in one download by registering
this provider configuration:

```yaml
- id: ReadInventory
  provider: XlsxReader
  params:
    - ranges:
        - {id: inventory, sheet: Inventory, range: autoFilter}
        - {id: heading, sheet: Summary, range: A4:A4}
      metadata: true
```

The batch result includes `byteCount` and `grids` keyed by range ID. Each
grid contains `sheet`, `range`, `rows`, and `types`. `metadata: true`
adds `workbook`, and can also be used by itself. The workbook inventory
contains ordered `sheetNames`, `sheetCount`, `definedNames` and
`definedNamesBySheet` (global scope uses `*`). Each defined name includes its
original XML `attributes`. The inventory also contains `sheets` keyed by
name and `charts` keyed by package part, with `chartCount`.

Each worksheet entry contains `kind`, `autoFilter`, `pane`, `hyperlinks`, its chart
part references, `maxRow`, and sparse `cells`, `types`, `formulas`, and
`errors` maps keyed by cell reference. Formula metadata records the observed
`expression` and `cachedValue` separately. Formula and error cells have
null data values and both are rejected inside a requested grid. Charts
contain `references`, `legendPresent`, and `series` with chart type,
direction, grouping, label, category and value references, and signed fill
presence. The reader records external hyperlink targets as text and never
follows them or evaluates formulas.
Standalone chart tabs appear in sheet order with `kind: chartsheet`, their
chart part references, and no cell grid.

With `metadata: true`, optionally set `metadataTextSheets: [Sheet1, Notes]`
to add `text` only to those worksheet entries. Names must match authored
worksheet names exactly; missing names, duplicate names (including case
variants), and chart tabs are rejected. Text joins non-null observed values
in row/column order with spaces, preserving strings and formatting native
numbers and booleans as text. Formula caches, error cells, and style-only
padding are excluded. Existing decoded values are reused; no additional
workbook read occurs. Duplicated text and separators count toward the same
16 MiB output text limit. Omitting this option preserves the existing output
and performs no text sorting or concatenation.

Merged cells intersecting a requested grid are unsupported. The workbook is
limited to 64 MiB before and after unpacking. The cumulative grid and
inventory output is limited to 100,000 cells and 16 MiB of decoded text,
including repeated ranges. Worksheets must have explicit ordered coordinates.
Selected tabs must resolve to distinct workbook parts. Shared strings have a
workbook-wide materialization limit of 2,000,000 rich-text run visits, charged
before each native read.
Reads without inventory decode only selected values and limit each selected
worksheet to 100,000 slots after expanding sparse rows and columns. A batch
without inventory reuses overlapping observations and limits native sheet,
row, and column lookup work to 2,000,000 comparisons per workbook. The
single-grid form keeps its existing lookup behavior. Inventory streams values,
skips style-only padding, and limits total padded row and column work to
2,000,000 slots. Requests with merged cells are limited to
1,000,000 merge intersection checks per workbook. Use a distinct UDF ID for
each registration with different worksheet options.

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
