# Slack service

- [Usage](#usage)
- [Endly service actions](#endly)
- [Credentials](#credentials)


## Usage

```bash
endly -r=test
```

[test.yaml](test.yaml)

```yaml
init:
  channel: '#serverless'
defaults:
  credentials: slack
pipeline:
  listen:
    action: slack:listen
    description: listen for incoming slack messages
    channel: $channel
  post:
    action: slack:post
    channel: $channel
    messages:
      - text: test is 1st test message
      - text: test is 2nd test message
  validate:
    action: slack:pull
    expect:
      - text: test is 1st test message
      - text: test is 2nd test message

```


<a name="endly"></a>
## Endly service action integration

Run the following command for slack service operation details:

```bash
endly -s=slack 
endly -s=slack -a=post
```


**Slack Service**

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- | 
| slack | post | post message to a slack channel | [PostRequest](contract.go) | [PostResponse](contract.go) | 
| slack | listen | listen for slack events to place then on pending validation queue| [ListenRequest](contract.go) | [ListenResponse](contract.go) | 
| slack | pull | pull/validate queued message | [PullRequest](contract.go) | [PullResponse](contract.go) | 


<a name="credentials"> </a>

## Credentials 

Generate encrypted endly credentials
- username: app/bot name
- password: app token

```go
endly -c=SECRET_NAME
ls -al ~/.secret/SECRET_NAME.json
```

where SECRET_NAME can be slack or any arbitrary credentials name