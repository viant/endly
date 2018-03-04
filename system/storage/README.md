** Storage Service**

Storage  service represents local or remote storage to provide unified storage operations.
Remote storage could be any cloud storage i.e. google cloud, amazon s3, or simple SCP or HTTP.
 

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| storage | copy | copy one or more resources from the source to target destination | [CopyRequest](service_storage_copy.go) | [CopyResponse](service_storage_copy.go) |
| storage | remove | remove or more resources if exsit | [RemoveRequest](service_storage_remove.go) | [RemoveResponse](service_storage_remove.go) |
| storage | upload | upload content pointed by context state key to target destination. | [UploadRequest](service_storage_copy.go) | [UploadResponse](service_storage_upload.go) |
| storage | download | copy source content into context state key | [DownloadRequest](service_storage_download.go) | [DownloadResponse](service_storage_download.go) |

