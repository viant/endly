#Storage Service - storage automation and testing

This service uses [Abstract File Storage](https://github.com/viant/afs).


- [Introduction](#introduction)
- [Usage](#usage)
- [Data copy](#data-copy)
  * [Multi asset copy](#multi-asset-copy)
  * [Expanding transferred data](#expanding-transferred-data)
  * [Expanding conditionally transferred data](#expanding-conditionally-transferred-data)
  * [Compressing transferred data](#compressing-transferred-data)  
  * [Archive transfer](#archive-transfer)
  * [Archive substitution transfer](#archive-substitution-transfer)
  * [Assets udf transformation](#assets-udf-transformation)
- [Listing location content](#listing-location-content)
  * [Applying browsing basic criteria](#applying-browsing-basic-criteria)
  * [Applying browsing time criteria](#applying-browsing-time-criteria)
- [Data upload](#data-upload)  
  * [Customer key data encryption](#customer-key-data-encryption)
  * [Dynamic conifg/state](#dynamic-configstate-upload)
- [Data validation](#data-validation)
- [Generating file](#generating-file)

## Introduction

This service provides local, remote and cloud storage utilities, 
for simple build automation, test preparation and verification.


To check all storage service methods run

```endly -s=storage```

To check individual method contract run:

```endly -s=storage:method```

For example to check all copy method contract options run:

```endly -s=storage:copy```


## Usage

You can integrate storage service with unit, integration and end to end tests.
For example to copy assets from local file system to Google Storage using automation workflow you can use the following:

```endly cp```

[@cp.ymal](usage/copy/cp.yaml)
```yaml
pipeline:
  copy:
    action: storage:copy
    source:
      URL: /tmp/folder
    dest:
      URL: s3://mybucket/data
      credentials: aws-e2e
    
```



to use API you can use the following snippet:

```go

import (
	"github.com/viant/endly"
	"github.com/viant/endly/system/storage"
	"github.com/viant/endly/system/storage/copy"
	"github.com/viant/toolbox/url"

	"log"
)

func main() {
	request := storage.NewCopyRequest(nil, copy.New(url.NewResource("/tmp/folder"), url.NewResource("s3://mybucket/data", "aws-e2e"), false, true, nil))
	response := &storage.CopyResponse{}
	err := endly.Run(nil, request, response)
	if err != nil {
		log.Fatal(err)
	}
}
```


## Data copy

To copy data from source to destination you can use the following workflow 
You can optionally specify prefix, suffix or filter expression that will match assets in a source location. 

[@copy.yaml](usage/copy/copy.yaml)
```yaml
init:
  bucket: e2etst

pipeline:
  copy:
    action: storage:copy
    suffix: .txt
    source:
      URL: data
    dest:
      credentials: gcp-e2e
      URL: gs://$bucket/copy/data
  list:
    action: storage:list
    source:
      credentials: gcp-e2e
      URL: gs://$bucket/copy/data

```

### Multi asset copy

To copy only a few asset from source location to destination you can use the following workflow:


[@multi_cp.yaml](usage/copy/multi_cp.yaml)

```yaml
pipeline:
  copy:
    action: storage:copy
    source:
      URL: data/
    dest:
      URL: /tmp/data/multi/
    assets:
      'lorem1.txt': 'lorem1.txt'
      'lorem2.txt': renamedLorem2.txt
```


### Expanding transferred data

When data is transferred between source and destination you can set expand flag to dynamically evaluate and workflow state variable, 
or you can provide a replacement map.

For example to substitute $expandMe expression and Lorem fragment when copying data from [@data/lorem2.txt](usage/copy/data/lorem2.txt) 
to destination you can use the the following workflow.


[@expanded_cp.yaml](usage/copy/expanded_cp.yaml)
```yaml
init:
  bucket: e2etst
  expandMe: '#dynamicly expanded#'

pipeline:
  copy:
    action: storage:copy
    expand: true
    replace:
      Lorem: blah
    source:
      URL: data
    dest:
      credentials: gcp-e2e
      URL: gs://$bucket/copy/modified

  list:
    action: storage:list
    content: true
    source:
      credentials: gcp-e2e
      URL: gs://$bucket/copy/modified

```

**expand** attribute instruct runner to expand any state variable matching '$'expression
**replace** defines key value pairs for basic text replacements.


### Expanding conditionally transferred data

When transferring and expanding data, you can also provide matcher expression to expand only specific asset.

For example to apply substitution only to file with _suffix_ lorem2.txt you can use expandif node with suffix attribute.

[@expandedif_cp.yaml](usage/copy/expandedif_cp.yaml)
```yaml
init:
  bucket: e2etst
  expandMe: '#dynamicly expanded#'

pipeline:
  copy:
    action: storage:copy
    expandIf:
      suffix: lorem2.txt
    expand: true
    replace:
      Lorem: blah
    source:
      URL: data
    dest:
      credentials: gcp-e2e
      URL: gs://$bucket/copy/filter_modified

  list:
    action: storage:list
    content: true
    source:
      credentials: gcp-e2e
      URL: gs://$bucket/copy/filter_modified

```


### Compressing transferred data

When dealing with large files amount you can compress them on the source location, transfer archive
and uncompress on the destination location with **compress** flag set.

[@compressed_cp.yaml](usage/copy/compressed_cp.yaml)
```yaml
pipeline:
  copy:
    action: storage:copy
    compress: true
    source:
      URL: data/
    dest:
      URL: /tmp/compressed/data

```

Currently this option is only supported with local or scp transfer type.

### Archive transfer

When transferring data, destination can be any supported by [Abstract File Storage](https://github.com/viant/afs) URL.


For example to copy local folder to zip archive on Google Storage you can run the following workflow.

[@archive.yaml](usage/copy/archive.yaml)
```yaml
init:
  bucket: e2etst

pipeline:
  copy:
    action: storage:copy
    source:
      URL: data
    dest:
      credentials: gcp-e2e
      URL: gs:$bucket/copy/archive/data.zip/zip:///data
  listStorage:
    action: storage:list
    source:
      credentials: gcp-e2e
      URL: gs://$bucket/copy/archive

  listArchive:
    action: storage:list
    source:
      credentials: gcp-e2e
      URL: gs:$bucket/copy/archive/data.zip/zip:///

```

When _gs://$bucket/copy/archive/data.zip_ archive does not exists it will be created on the fly, if already does
the source assets will be appended or replaced to existing archive.


### Archive substitution transfer

To dynamically append/replace asset with dynamic data substitution you can use the following workflow.

[@archive_update.yaml](usage/copy/archive_update.yaml)
```yaml
init:
  changeMe: this is my secret

pipeline:
  copy:
    action: storage:copy
    source:
      URL: app/app.war
    dest:
      URL: /tmp/app.war

  updateArchive:
    action: storage:copy
    expand: true
    source:
      URL: app/config.properties
    dest:
      URL: file:/tmp/app.war/zip://localhost/WEB-INF/classes/

  checkUpdate:
    action: storage:download
    source:
      URL: file:/tmp/app.war/zip://localhost/WEB-INF/classes/config.properties
    destKey: config

  info:
    action: print
    message: $checkUpdate.Payload
```


### Assets udf transformation.

When transferring data you can apply transformation to each transferred asset using pre defined UDF:
For example to apply Gziper udf to copied file you can you the following:

[@udf.yaml](usage/copy/udf.yaml)
```yaml
init:
pipeline:
  upload:
    action: storage:copy
    source:
      URL: data/lorem1.txt
    udf: GZipper
    dest:
      URL: /tmp/lorem.txt.gz
```


## Listing location content

To list location content you can use storage service list method

To list recursively gs://$bucket/somepath content you can use the following:

[@list.yaml](usage/list/list.yaml)
```yaml
init:
  bucket: myBucket
pipeline:

  list:
    action: storage:list
    recursive: true
    content: false
    source:
      credentials: gcp-e2e
      URL: gs://$bucket/somepath

```

when **content** attribute is set, list operation downloads asset content.

### Applying browsing basic criteria


When listing content you can specify a [Basic](https://github.com/viant/afs/blob/master/matcher/basic.go) matcher criteria.

[@filter.yaml](usage/list/filter.yaml)
```yaml
init:
  bucket: e2etst
pipeline:

  list:
    action: storage:list
    recursive: true
    match:
      suffix: .txt
    source:
      credentials: gcp-e2e
      URL: gs://$bucket/

```

### Applying browsing time criteria

In some situation exact file name may be dynamically generated with UUID generator, so in that case
you can use ```updatedAfter``` or ```updatedBefore``` [TimeAt expression](https://github.com/viant/toolbox/#time-utilities)
to matched desired asset.

For example the following workflow create assets in Google Storage and then lists it by time expression.

[@time_filter.yaml](usage/list/time_filter.yaml)
```yaml


```yaml
init:
  i: 0
  bucket: e2etst
  baseURL: gs://$bucket/timefilter
  data: test
pipeline:

  batchUpload:
    upload:
      init:
        _: $i++
      action: storage:upload
      sleepTimeMs: 1200
      sourceKey: data
      dest:
        credentials: gcp-e2e
        URL: ${baseURL}/subdir/file_${i}.txt
    goto:
      when: $i < 3
      action: goto
      task: batchUpload

  list:
    action: storage:list
    recursive: true
    logging: false
    content: true
    match:
      suffix: .txt
      updatedAfter: 2secAgo

    source:
      credentials: gcp-e2e
      URL: $baseURL

    message: $AsString($list.Assets)
```


## Data upload

Data upload enables to upload workflow state directly to desired storage location.


### Customer key data encryption


[@custom_key.yaml](usage/upload/custom_key.yaml)
```yaml
init:
  data: $Cat('lorem.txt')
  bucket: e2etst
  customerKey:
    key: this is secret :3rd party phrase

pipeline:
  upload:
    action: storage:upload
    sourceKey: data
    dest:
      URL: gs://$bucket/secured/lorem.txt
      credentials: gcp-e2e
      customKey: $customerKey
  list:
    action: storage:list
    source:
      URL: gs://$bucket/secured/
      credentials: gcp-e2e
  download:
    action: storage:download
    source:
      URL: gs://$bucket/secured/lorem.txt
      credentials: gcp-e2e
      customKey: $customerKey
  info:
    action: print
    message: 'Downloaded: $AsString(${download.Payload})'
```


### Dynamic config/state upload



[@dynamic.yaml](usage/upload/dynamic.yaml)
```yaml
init:
  settings: $Cat('settings.json')
  settingsMap: $AsMap('$settings')
  config:
    key1: val1
    key2: val2
    featureX: ${settingsMap.featureX}


pipeline:
  info:
    action: print
    message: $AsString('$config')

  dynamic:
    init:
      cfg: $AsJSON('$config')
    action: storage:upload
    sourceKey: cfg
    dest:
      URL: /tmp/app.json
```

## Data validation

The following service operations provide validation integration by 'expect' attriubte
 - storage:list
 - storage:exists
 - storage:download
 
When defining expect attribute you can use rule based [assertly validation expressioa](https://github.com/viant/assertly/#validation)

For example to dynamically uncompress data/events.json.gz to 
perform [structured data](usage/download/data/expect.json) validation you can 
use the following workflow:

[@download.yaml](usage/download/download.yaml)

```yaml
init:
  expect: $Cat('data/expect.json')

pipeline:
  check:
    action: storage:download
    udf: UnzipText
    source:
      URL: data/events.json.gz
    expect: $expect
```


To validate if files exists you can you the following workflow:

[@exists.yaml](usage/exists/exists.yaml)
```yaml
pipeline:
  check:
    action: storage:exists
    assets:
      - URL: data/f1.txt
        credentials: localhost
      - URL: data/f2.txt
      - URL: data/f3.txt
      - URL: gs://blach/resource/assset1.txt
        credentials: gcp-e2e

    expect:
      'data/f1.txt': true
      'data/f2.txt': false
      'data/f3.txt': true
      'gs://blach/resource/assset1.txt': false
```



### Generating file

