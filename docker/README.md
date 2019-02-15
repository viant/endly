# Endly docker image

[Endly docker repository](https://cloud.docker.com/repository/docker/endly/endly)

- alpine based: endly/endly:latest-alpine3.8
- ubuntu based: endly/endly:latest-ubuntu16.04 (for serverless testing)



### Getting started

```bash
mkdir -p ~/e2e
mkdir -p ~/.secret

docker run --name endly -v /var/run/docker.sock:/var/run/docker.sock -v ~/e2e:/e2e -v ~/e2e/.secret/:/root/.secret/ -p 7722:22  -d endly/endly:latest-ubuntu16.04  
ssh root@127.0.0.1 -p 7722 ## password is dev
endly -v

### create secrets for endly with root/dev credentials (one off)
endly -c=dev
ls -al 

## create hello.yaml

endly -r=hello name=Endly

```



@hello.yaml
```yaml
init:
  target:
    URL: ssh://127.0.0.1
    credentials: dev
  name: $params.name    
pipeline:
  task1:
    action: print
    message: Hello World $name
  task2:
    action: exec:run
    target: $target
    commands:
      - echo 'Hello World $name'
```

[Endly introduction](http://github.com/adrianwit/endly-introduction)