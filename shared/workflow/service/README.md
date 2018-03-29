#### Services

**Workflow services:**

| Name | Description |
| ---- | --- |
| [aerospike](aerospike/) | start/stop aerospike,  test and wait if it is ready to use |
| [mysql](mysql/) | start/stop mysql,  test and wait till it is ready to use, export, imports data |
| [postgress](pg/) | start/stop postgresSQL, test and wait till it is ready to use |
| [memcached](memcached/) | start/stop memcached |
| [mongoDB](mongo/) | start/stop mongoDB, test and wait till it is ready to use  |
 

##### usage:
 
```bash
# call workflow with task start/stop
endly -w=service/mysql -t=start
```
 
