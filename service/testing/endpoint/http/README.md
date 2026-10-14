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

Calling `listen` again on the same port reuses the existing recording by default.
For a new fixture in a retained MCP session, set `replace: true`. Endly loads and
validates the new recording before replacing it on the existing socket, resets
replay positions, and applies the new `rotate`, `indexKeys`, and file templates.
The bind host must remain the same. A failed load leaves the previous recording
available; session cleanup still closes the listener.

```yaml
action: http/endpoint:listen
host: 127.0.0.1
port: 8080
replace: true
rotate: true
baseDirectory: /next_fixture/
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

### Identity-provider mocks

`indexKeys: [Method, URL, Header.Authorization]` matches separate identities
without comparing changing bodies or incidental headers. Header names are case
insensitive. Unmatched requests return 404 without disclosing recorded headers.
Responses for the same key are replayed in file order. `rotate: false` returns
404 after exhaustion; `rotate: true` wraps to the first response. `append`
replaces matching keys and retains other keys, including earlier appends.

Use `host: 127.0.0.1` for a local mock. Omitting `host` preserves the existing
all-interface binding. `shutdown` releases the port and permits a later `listen`
on the same port. Closing the Endly context also stops its mock servers.

This service replays HTTP contracts; it does not implement an OAuth issuer or
role database. Generate synthetic signed tokens and captured request/response
files in the test fixture. The application should still verify identity and
use its actual authorization engine for decisions.
