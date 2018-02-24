# Transformer application - simple data copy from one datastore to another, i.e. (aerospike to mysql)

This application was build to provide end to end testing example of etl application.

The end to end test provide use cases to test aerospike to mysql backup with and without transformation.

Prerequisites:

Enable ssh logic you your use on your machine (on osx System Preference / Sharing / Remote Login )
 
Install [docker](https://docs.docker.com/engine/installation/) service


[Download endly and secret for your platofrm](https://github.com/viant/endly/releases/)

Or alternatively build binary from scratch following above instruction:

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

Generate credentials file used by the workflow.
```text
endly -c=CREDENTIAL_FILENAME
```
Command endly -c generates a file that store blowfish encrypted password in $HOME/.secret/ directory.

Provide a username and password to login to your box.
```text
mkdir $HOME/.secret
ssh-keygen -b 1024 -t rsa -f id_rsa -P "" -f $HOME/.secret/id_rsa
cat $HOME/.secret/id_rsa.pub >  ~/.ssh/authorized_keys 
chmod u+w authorized_keys

endly -c=localhost -k=~/.secret/id_rsa.pub
```
```

Verify that secret file were created
```text
cat ~/.secret/localhost.json
```



Provide  **root** as username and non empty password for docker mysqladmin
```text
endly -c=mysql
```


Verify that secret file were created
```text
cat ~/.secret/localhost.json
```


If toy build endly from scrach check that **'endly'** binary is created in $GOPATH/bin directory as result of 
'go get -u github.com/viant/endly/endly'



#### Run transformer workflow

Run the following command:

```text
cd $GOPATH/src/github.com/viant/endly/example/ws/etl/transformer/endly/
endly
```


#Troubleshooting

to check you aerospike just run

docker exec -it db4_aerospike aql
SHOW sets;


docker exec -it db3_mysql mysql
show tables;


  
