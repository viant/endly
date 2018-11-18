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


### Predefined UDF Providers

| Provider | Arguments | 
|---|---| 
| ProtoReader |  schemaFile, messageType, importPath |
| ProtoReader | schemaFile, messageType, importPath |
| AvroWriter | avroSchema/URL, compression |


### Service actions

**UDF Service**

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| udf | register | register custom UDF with udf provider | [RegisterRequest](service_contract.go) | [RegisterResponse](service_contract.go) |
