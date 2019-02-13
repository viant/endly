## Kubernetes service

Work in progress ... !!!!


This service is *k8s.io/client-go/kubernetes.Clientset proxy 

It defines helper method to deal with basic operation in a friendly way.


To check all supported method run
```bash
     endly -s='kubernetes'
```

or to check service contract ```endly -s='kubernetes' -a=ACTION```

```bash
     endly -s='kubernetes' -a=get
```


## Usage

## Get k8s resource info


```bash
endly -run='kubernetes:get' kind=pod
```

```bash
 endly -run=kubernetes:get kind=pod labelSelector="run=load-balancer-example"
```

```bash
endly -run=kubernetes:get kind=endpoints 
```



## Create resources
Create resource(s) with create API method call.

1. Create with external resource:
    ```bash
    endly -run='kubernetes:create' URL=test/deployment.yaml
    ```
    
2. Create workflow
    * [@create.yaml](test/create.yaml) ```endly -r=create```
    ```yaml
    pipeline:
      createNginx:
        action: kubernetes:create
        apiVersion: apps/v1
        kind: Deployment
        metadata:
          labels:
            run: nginx
          name: nginx
        spec:
          replicas: 1
          selector:
            matchLabels:
              run: nginx
          template:
            metadata:
              labels:
                run: nginx
            spec:
              containers:
                - image: nginx
                  imagePullPolicy: Always
                  name: nginx
              restartPolicy: Always
    ```

## Apply resources state
Create resource(s) with create API method call or apply patch.

1. Apply with external resource:
    ```bash
    endly -run='kubernetes:apply' URL=test/deployment.yaml
    ```

2. Apply workflow
    * [@apply.yaml](test/apply.yaml) ```endly -r=apply```
    ```yaml
    pipeline:
      applyNginx:
        action: kubernetes:apply
        URL: someURL.yaml
    ```

## Run - crete resource from template
1. Start a single instance (apps/v1.Deployment template)
- Start a single instance of nginx cli.
    ```bash
    endly -run='kubernetes:run' name=nginx image=nginx  port=80 
    ```
- Start a single instance of hazelcast, expose port 5701 and set environment variables "DNS_DOMAIN=cluster" and "POD_NAMESPACE=default" in the container.
    ```bash
    endly -run='kubernetes:run' name=hazelcast image=hazelcast port=5701 env.DNS_DOMAIN=cluster env.POD_NAMESPACE=default
    ```
- Start a single instance of hazelcast and set labels "app=hazelcast" and "env=prod" in the container.
    * [@run_single.yaml](test/run_single.yaml) ```endly -r=run_single```
    ```yaml
    pipeline:
      runHazelcast:
        action: kubernetes:run
        name: hazelcast
        image: hazelcast
        labels:
          app: hazelcast
          env: prod
    ````
2. Start a replicated instance (apps/v1.Deployment template)
    * [@run_replicas.yaml](test/run_replicas.yaml) ```endly -r=run_replicas```
    ```yaml
    pipeline:
      runNgix:
        action: kubernetes:run
        name: nginx
        image: nginx
        replicas: 5
    ````
3. Start single pod instance (v1.Pod template)
    * [@run_pod.yaml](test/run_pod.yaml) ```endly -r=run_pod```
    ```yaml
    pipeline:
      runPod:
        action: kubernetes:run
        name: nginx
        image: nginx
        restartPolicy: Never
    ````
4. Start the perl container to compute π to 2000 places and print it out. (batch/v1.Job template)
    * [@run_job.yaml](test/run_job.yaml) ```endly -r=run_job```
    ```yaml
    pipeline:
      runJob:
        action: kubernetes:run
        name: pi
        image: perl
        restartPolicy: OnFailure
        commands:
          - "perl -Mbignum=bpi -wle 'print bpi(2000)'"
    ```
5. Start the cron job to compute π to 2000 places and print it out every 1 minutes.
    * [@schedule.yaml](test/schedule.yaml) ```endly -r=schedule```
    ```yaml
    pipeline:
      runJob:
        action: kubernetes:run
        name: pi
        image: perl
        schedule: 0/1 * * * ?
        commands:
          - "perl -Mbignum=bpi -wle 'print bpi(2000)'"
    ```

## Expose
Expose resource(s) port via service port.

1. Create a service for a explicit resource and port
    ```go
    endly -run=kubernetes:expose resource=Deployment/nginx port=8080 targetPort=80 type=NodePort
    ```
2. 
    * [@expose.yaml](test/expose.yaml) ```endly -r=expose```
    ```yaml
    pipeline:
      applyNginx:
        action: kubernetes:expose
        name: myService
        resource: Deployment/myApp
    ```
## Copy

## Logs

## Delete

## Config Maps

## Secrets


## Global contract parameters
- context
- namespace
