## SMTP Endpoint Service

This service allows email message validation.
Listen operation starts SMTP endpoint that places all incoming messages to validation queue.
Assert operation validates mail messages.


| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- | 
| smtp/endpoint | listen | start accepting mails | [ListenRequest](contract.go) | [ListenResponse](contract.go) | 
| smtp/endpoint  |  assert | perform validation on provided expected user message  | [AssertRequest](contract.go) | [AssertResponse](contract.go) | 


### Usage

```bash
endly -r=test.yaml
```


[@test.yaml](test/test.yaml)
```yaml
pipeline:
  listen:
    action: smtp/endpoint:listen
    port: 1465
    enableTLS: true
    certLocation:
    users:
      - username: bob
        credentials: e2eendly

  send:
    action: smtp:send
    target:
      URL: smtp://localhost:1465
      credentials: e2eendly
    mail:
      from: tester@localhost
      to:
        - bob@localhost
      subject: test subject
      body: this is test body

  assert:
    action: smtp/endpoint:assert
    expect:
      - user: bob
        message:
          Subject: test subject2
          Body: /test body/
```


### Using SSL/TLS

When enabling SSL/TLS for testing you can use the following command to generate self describing cert:

```bash
openssl req -newkey rsa:2048 -new -nodes -x509 -days 3650 -keyout key.pem -out cert.pem
```