# Logger - simple http request event logger

This application was build to provide end to end testing example of etl application.

The end to end test provide uses to test aerospike to mysql backup with and without transformation.

Prerequisites:

Enable ssh logic you your use on your machine (on osx System Preference / Sharing / Remote Login )
 
Install [docker](https://docs.docker.com/engine/installation/) service
Install [go lang](https://golang.org/doc/install) version 1.8+

after installing go run the following command

```text
mkdir -p ~/Projects/go
export GOPATH=~/Projects/go
go get -u github.com/viant/endly
go get -u github.com/viant/endly/endly
go get -u github.com/viant/toolbox/secret

export PATH=$PATH:$GOPATH/bin
```

Generate secret keys with a credential that endly will use to run the workflows.
(**secret** binary should be compiled and build as result of get -u github.com/viant/toolbox/secret into GOPATH/bin)
Secret generate a file that store blowfish encrypted credential in $HOME/.secret/ directory.


Provide a user name and password to login to your box.
```text
secret scp
```
```

Verify that secret file were created
```text
cat ~/.secret/scp.json
```


Check that **'endly'** binary is created in $GOPATH/bin directory as result of 
'go get -u github.com/viant/endly/endly'


#### Run logger workflow

Run the following command:

```text
cd $GOPATH/src/github.com/viant/endly/example/ws/rt/elogger/endly/
endly
```
