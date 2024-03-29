# Amazon Relational Database Service service

This service is github.com/aws/aws-sdk-go/service/rds.RDS proxy 

To check all supported method run
```bash
    endly -s="aws/rds"
```

To check method contract run endly -s="aws/cloudwatch" -a=methodName
```bash
    endly -s=aws/rds:describeDBInstances
```

#### Usage:

###### Extracting RDS endpoint

```endly -r=info```
[@info.yaml](info.yaml)
```yaml
pipeline:
  getInfo:
    action: aws/rds:describeDBInstances
    credentials: aws-e2e
    dbinstanceidentifier: db
    logging: false
    post:
      dbIP: $DBInstances[0].Endpoint.Address

  showEndpint:
    action: print
    message: 'RDS endpoint: $dbIP'
```
