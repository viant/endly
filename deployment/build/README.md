**Build service**

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| build | load | load BuildMeta for the supplied resource | [LoadMetaRequest](service_contract.go) | [LoadMetaResponse](service_contract.go)  |
| build | register | register BuildMeta in service repo | [RegisterMetaRequest](service_contract.go) | [RegisterMetaResponse](service_contract.go)  |
| build | build | Run build for provided specification | [Request](service_contract.go) | [Response](service_contract.go)  |

