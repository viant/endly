### Build workflow


Upload assets, install dependencies, build app, download built app 

#### _Pipeing workflows/actions_

```bash
endly.go -p=run
```

@run.yaml
```yaml
params:
  app: echo
  secrets:
    localhost: localhsot
  target:
    URL:ssh://127.0.0.1/
    Credentials:localhost
  sdk: go:1.8
pipeline:
  build:
    workflow: docker/build
    origin:
      URL: http://github.com/adrianwit/echo
    upload:
      $Pwd(test/pipeline/build.yaml): /$app/
    commands:
      - apt-get -y install telnet
      - cd /$app
      - go build -o echo
    download:
      /$app: /tmp/$app/
  test:
    action: workflow:print
    message: testing app ...
```
