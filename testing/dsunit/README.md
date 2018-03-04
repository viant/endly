**Datastore services**


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


