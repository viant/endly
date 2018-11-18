# User defined function

**User defined function** is a way to transform data on fly to be consumed in the desired way.

_UDF_ implement the following function signature `func(source interface{}, state data.Map) (interface{}, error)`



- [Usage](#usage)
- [Build in UDF](#buildin)
- [Custom UDF](#custom)
- [UDF Providers](#providers)

### Usage:
<a name="usage"></a>
**Endly Action** 

Some _endly_ service actions support UDF as part of their request contract. 


Imagine that you want to test REST service which accepts application/octet-stream content type with avro payload. 
The following HTTP request definition uses Avro UDFs.



```json
{
  "Requests": [
    {
      "Method": "post",
      "URL": "http://127.0.0.1:8987/xxx?access_key=abc",
      "Header":{
        "Content-Type":["application/octet-stream"]
      },
      "RequestUdf": "UserAvroWriter",
      "ResponseUdf": "AvroReader",
      "JSONBody": {
        "ID":1,
        "Desc":"abc"
      }
    }
  ],
  "UdfProviders": [
    {
      "Id": "UserAvroWriter",
      "Provider": "AvroWriter",
      "Params": [
        "{\"type\": \"record\", \"name\": \"user\", \"fields\": [{\"name\": \"ID\",\"type\":\"int\"},{\"name\": \"Desc\",\"type\":\"string\"}]}"
      ]
    }
  ],
  "Expect": {
      "Responses": [
        {
          "Code": 200,
          "JSONBody": {
            "Status": "ok",
            "Data": {
              "ID":1,
              "Desc":"abc"
            }
          }
        }
      ]
    }
}
```

**Data Substitution**
Another way of consuming UDF is via reference in data itself, in that case just use $ in front of UDF name followed by (parameters).
In data substitution case if UDF returns error data will NOT be expanded with corresponding UDF.

@user.json
```json
{
  "Id":1111,
  "Name":"User A",
  "DataOfBirth":"$Dob([14,2,2,\"yyyy\"])"
}

```


###  Build in UDF 
<a name="buildin"></a>


**Defined in [endly project](./../../udf/udf.go)**

| UDF | Description | Inline Example |
|---|----|----|
| DateOfBirth | provides formatted date of birth, it take  desired age, optionally month, day and timeformat | $Dob([yeaysAgo,monthsAgo,daysAgo,"yyyy"]) |
| URLJoin | joins base URL and URI path | $URLJoin($baseURL, $URI) |
| Hostname | extracts host from URL | $Hostname($URL) |
| AvroReader | Avro reader | n/a | 

**Defined in [dsunit project](./../../testing/dsunit/udf.go)**

| UDF | Description |
|---|----|
| AsTableRecords | udf converting []*DsUnitTableData into map[string][]map[string]interface{} (used by prepare/expect dsunit service), as table record udf provide sequencing and random id generation functionality for supplied data . | 


**Defined in [neatly project](https://github.com/viant/neatly/blob/master/udf.go)**


| UDF | Description | Inline Example |
|---|----|----|
| AsMap | self explanatory | $AsMap($var) |
| AsInt |self explanatory | | $AsMap($var) |
| AsFloat |self explanatory | $AsFloat($var) |
| AsBool  | self explanatory |$AsBool($var) |
| HasResource | returns true if external resource exists | $HasResource(/opt/location/) |
| Md5 | generates md5 for provided parameter | $Md5($var) | 
| WorkingDirectory | returns working directory joined with supplied sub path,  '../' is supported | $WorkingDirectory(../) |
| LoadNeatly | loads neatly document as data structure. | n/a | 
| Length or Len | returns length of slice, map or string | $Len($failures) | 
| FormatTime | takes two arguments, date or data literal like 'now' or '2hoursahead', followed by java style date format | $FormatTime(["now","yyyyMMdd"]) | 
| Zip | takes []byte or string to compress it. | n/a | 
| Unzip | takes []byte to uncompress it into []byte. | n/a | 
| UnzipText | takes []byte to uncompress it into string.| n/a | 
| Markdown | generate HTML for suppied markdown| $Markdown($md) | 
| Cat | returns content of supplied filename | $Cat("/etc/hosts") |  
| Increment | increments state key value with supplied delta  | $Increment(['counterKey', -2]), | 



### Custom UDF
<a name="custom"></a>

In order to define a custom UDF you would have to brnach your endly executor using the following snippet.

@endly.go

```go
package main;

import	"github.com/viant/endly/bootstrap"
import	"github.com/viant/toolbox/data"
import	"github.com/viant/endly"


func MyUDF(source interface{}, state data.Map) (interface{}, error) {
    return nil, nil
}


func init() {
	endly.UdfRegistry["MyUDF"] = MyUDF
}


func main() {
		bootstrap.Bootstrap()
}

```

And to build it.

```bash
go build endly.go
```

### UDF Providers
<a name="providers"></a>

UDF provider has ability to create an instance of UDF with custom settings. 
Imagine UDF transforming data with avro codec, without UDF provider you would have to 
create your custom UDF for each data schema with custom endly build.
This can be avoided  with UDF provider, in this case you would register an avro writer witch specific data schema.

To register custom UDF use [udf](./../../udf) service with register action.

i.e: 

```go
endly -r=reqister
```

@register.yaml
```yaml
pipeline:
    register-udf:
      action: udf:register
      udfs:
        - id: ProfileToProto
          provider: ProtoWriter
          params:
            - /Project/proto/up.proto
            - Profile
        - id: ProtoToProfile
          provider: ProtoReader
          params:
            - /Project/proto/ip.proto
            - Profile
```
