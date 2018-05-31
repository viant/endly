#Storage Service

Storage  service represents local or remote storage to provide unified storage operations.
Remote storage could be any cloud storage i.e. google cloud, amazon s3, or simple SCP or HTTP.


<a name="endly"></a>

## Endly inline workflow

```bash
endly -r=copy
```


@copy.yaml
```yaml

pipeline:
  transfer:
    action: storage:copy  
    source:
      URL: s3://mybucket/dir
      credentials: aws-west
    dest:
      URL: scp://dest/dir2
      credential: dest
    assets:
      file1.txt:
      file2.txt: renamedFile2      

```


## Endly workflow service action

Run the following command for storage service operation details:

```bash

endly -s=storage

endly -s=storage -a=copy
endly -s=storage -a=remove
endly -s=storage -a=upload
endly -s=storage -a=download

```
 


| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| storage | copy | copy one or more asset from the source to destination | [CopyRequest](service_storage_copy.go) | [CopyResponse](service_storage_copy.go) |
| storage | remove | remove or more resources if exsit | [RemoveRequest](service_storage_remove.go) | [RemoveResponse](service_storage_remove.go) |
| storage | upload | upload content pointed by context state key to target destination. | [UploadRequest](service_storage_copy.go) | [UploadResponse](service_storage_upload.go) |
| storage | download | copy source content into context state key | [DownloadRequest](service_storage_download.go) | [DownloadResponse](service_storage_download.go) |


Storage service uses undelying [Storage Service](https://github.com/viant/toolbox/tree/master/storage)




## Asset copy

Copy request provides flexible way of transferring  assets from source to destination.

Make sure that when using cloud or other more specific URI scheme, the corresponding [imports](https://github.com/viant/toolbox/tree/master/storage#import) are in place.

Copy operation provides two way for asset content substitution:
1) Replace - simple brute force key value pair replacement mechanism
2) $ expression substitution from context.State()  mechanism



```go


    import   "github.com/viant/endly/storage"
    import _ "github.com/viant/toolbox/storage/aws"
    import   "github.com/viant/endly"
    import   "log"


    func copy() {
    	
    	var manager = endly.New()
    	var context := manager.NewContext(nil)
    	var s3CredentialLocation = ""
    	var request = storage.NewCopyRequest(nil, NewTransfer(url.NewResource("s3://mybucket/asset1", s3CredentialLocation), url.NewResource("/tmp/asset1"), false, false, nil))
    	err := endly.Run(context, request, nil)
    	if err != nil {
    		log.Fatal(err)
    	}
    }


```

**Loading request from URL**

[CopyRequest](service_contract.go) can be loaded from URL pointing either to JSON or YAML resource.

```go
    copyReq1, err := storage.NewCopyRequestFromURL("copy.yaml")
    copyReq2, err := storage.NewCopyRequestFromURL("copy.json")

```


**Single asset transfer**


@copy.yaml

```yaml
source:
  URL: mem://yaml1/dir
dest:
  URL: mem://dest/dir2
``` 



**Multi asset transfer with asset** 
In this scenario source and dest are used to build full URL with assets


@copy.yaml

```yaml
source:
  URL: mem://yaml1/dir
dest:
  URL: mem://dest/dir2
assets:
  file1.txt:
  file2.txt: renamedFile2  
```

**Multi asset transfer with transfers and compression**
 
@copy.yaml

```yaml
transfers:
- expand: true
  source:
    url: file1.txt
  dest:
    url: file101.txt
- source:
    url: largefile.txt
  dest:
    url: file201.txt
  compress: true
```

**Single asset transfer with replace data substitution**


@copy.json
```json
{
  "Source": {
    "URL": "mem://yaml1/dir"
  },
  "Dest": {
    "URL": "mem://dest/dir2"
  },
  "Compress":true,
  "Expand": true,
  "Replace": {
    "k1": "1",
    "k2": "2"
  }
}
```



**Single asset transfer with $expression data substitution**


```go

  func copy() {
    	
    	var manager = endly.New()
    	  
    	var context := manager.NewContext(nil)
    	var state = context.State()
    	
    	state.PutValue("settings.port", "8080")
    	state.PutValue("settings.host", "abc.com")
    	
    	var request = storage.NewCopyRequest(nil, NewTransfer(url.NewResource("myconfig.json"), url.NewResource("/app/config/"), false, true, nil))
    	err := endly.Run(context, request, nil)
    	if err != nil {
    		log.Fatal(err)
    	}
    }


```

@myconfig.json
```json
{
    "Port":"$settings.port",
    "Host":"$settings.host"
}

```


**Single asset transfer with custom copy handler using an UDF (User defined function). Below is an example to use custom UDF CopyWithCompression that gzips target file**


```json
  {
    "Transfers": [
      {
        "Source": {
          "URL": "s3://mybucket1/project1/Transfers/",
          "Credentials": "${env.HOME}/.secret/s3.json"
        },
        "Dest": {
           "URL": "gs://mybucket2/project1/Transfers/",
            "Credentials": "${env.HOME}/.secret/gs.gz"
        }
      },
      "CopyHandlerUdf": "CopyWithCompression"
    ]
  }


```