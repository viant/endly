# External Echo application test 

This application provides example of 3rd application build deployment and test automation with docker


1) This application binary is built inside docker container,
2) Build  binary is used by Dockerfile to build size optimized application docker image
3) Application image is deploy for testing
4) Test execution
5) Stopping application

If useRegistry: true, then endly login to docker registry, application is pushed to docker registry


Prerequisites:

Enable ssh logic you your use on your machine (on osx System Preference / Sharing / Remote Login )
 
Install [docker](https://docs.docker.com/engine/installation/) service

Download [endly](https://github.com/viant/endly/releases/)

Provide a username and password to login to your box.
```text
endly -c=localhost
```

Verify that secret file were created
```text
cat ~/.secret/localhost.json
```


#### Run echo docker workflow

Run the following command:

```text
git clone https://github.com/viant/endly
cd endly/example/echo
```

## run test with inline workflow[run](endly/run.yaml) request
```text
endly -r=run
```
