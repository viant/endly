**HTTP Endpoint Service**

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- | 
| http/endpoint | listen | listen on specified port to replay recorded HTTP conversation | [ListenRequest](service_contract.go) | [ListenResponse](service_contract.go) | 

This service enable capturing and replaying HTTP traffic to simulate 3rd party dependency.


### Capturing HTTP traffic

Capturing 3rd party http traffic

 sudo endly -u='http://targetURL'

 open you browser with various URL on localhost matching targetURL port
 

Capturing 3rd party secure http traffic

Make sure you have server cert and key, or you can generate self self-signed (x509) with the following

```bash
openssl genrsa -out server.key 2048
openssl ecparam -genkey -name secp384r1 -out server.key
openssl req -new -x509 -sha256 -key server.key -out server.crt -days 3650
```

sudo endly -u='https://some.domain.com'


### Starting testing endpoint with captured traffic

@listen.yaml

```yaml
port: 8080
rotate: true
baseDirectory: /recorded_traffic_location/
```

Start testing endpoint in standalone mode

```bash
endly -m=true  -w=action service='http/endpoint' action=listen request=@listen.yaml 
```

### Embeding endpoint within inline workflow

@inline.yaml

```yaml
pipeline:
  init:
    start-endpoint:
      action: http/endpoint:listen
      port: 8080
      rotate: true
      baseDirectory: /recorded_traffic_location/

```

Start your workflow

```bash
endly -r=inline -m=true
```
