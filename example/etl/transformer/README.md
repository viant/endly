# Transformer application - simple data copy from one datastore to another, i.e. (aerospike to mysql)

This application was build to provide end to end testing example of etl application.

The end to end test provide use cases to test aerospike to mysql backup with and without transformation.


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

Create 'mysql' secret credentials, provide  **root** as username and non empty password for docker mysqladmin
```text
endly -c=mysql
```



#### Run reporter webservice workflow

Run the following command:

```text
git clone https://github.com/viant/endly
cd endly/example/etl/transformer/endly/
```

## run test with [manager](endly/manager.csv) workflow:
```text
endly -w=manager
```
## run test with inline workflow[run](endly/run.yaml) request
```text
endly -r=run
```

## To check manager workflow tasks list
```text
endly -w=manager -t='?'
```



#Troubleshooting

to check you aerospike just run

docker exec -it mydb4 aql
SHOW sets;


docker exec -it mydb3 mysql
show tables;


  
