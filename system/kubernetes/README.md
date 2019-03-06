## Kubernetes service

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

## Apply - create or patch resources
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
## Delete
1. Delete resource
    ```bash
    endly -run='kubernetes:delete' kind=pod name=myPod
    ```
2. Delete resources 
    ```bash
    endly -run='kubernetes:delete' kind='*' name=myPod
    ```
3. Delete resource from file
    ```bash
    endly -run='kubernetes:delete' URL=resources.yaml
    ```

## Config Maps
1. Creating config maps with yaml resource file
    * endly -r=create_config
    * [@create_config.yaml](test/create_config.yaml)  [@config.yaml](test/config.yaml)
    ```yaml
    pipeline:
      createConfig:
        action: kubernetes:create
        URL: config.yaml
        expand: true
    ```
2. Creating config maps
    * ``endly -r=configmaps``
    * [@configmap](test/configmap.yaml)
    ```yaml
    pipeline:
      createConfig:
        action: kubernetes:create
        kind: ConfigMap
        apiVersion: v1
        metadata:
          name: examplecfg
        data:
          config.property.1: value1
          config.property.2: value2
          config.properties: |-
            property.1=value-1
            property.2=value-2
            property.3=value-3
        binaryData:
          foo: L3Jvb3QvMTAw
    ```

3. Creating config maps for folder/URL
    * ```endly -r=configmap_from_file```
    * [@configmap_from_file.yaml](test/configmap_from_file.yaml)
    ```yaml
    pipeline:
      createConfig:
        action: kubernetes:create
        kind: ConfigMap
        apiVersion: v1
        metadata:
          name: mycfg
        data: $AssetsToMap('config/')
        binaryData: $BinaryAssetsToMap('config/bin')
    ```

## Secrets
  
1. Creating secrets from literals
    * ``endly -r=raw_secrets``
    * [@raw_secrets](test/raw_secrets.yaml)
    ```yaml
    init:
      username: $Cat(somefile.txt)
      password: dev
    pipeline:
      setSecrets:
        action: kubernetes:apply
        apiVersion: v1
        kind: Secret
        metadata:
          name: my-secrets
        type: Opaque
        data:
          username: $Base64Encode($username)
          password: $Base64Encode($password)    
      ```
2. Creating secrets with endly secrets 
    * ``endly -r=endly_secrets``
    * [@endly_secrets](test/endly_secrets.yaml)
    ```yaml
    init:
      devSecrets: $secrets.dev
    pipeline:
      info:
        action: print
        message: $devSecrets.Data
    
      setSecrets:
        action: kubernetes:apply
        apiVersion: v1
        kind: Secret
        metadata:
          name: dev-secrets
        type: Opaque
        data:
          dev.json: $Base64Encode($devSecrets.Data)
          username: $Base64Encode($devSecrets.Username)
          password: $Base64Encode($devSecrets.Password)
    
    ```
3. Testing secrets without pipeline
    * ```endly kubernetes:get secrets kind=secret name=dev-secrets```
    * ```endly kubernetes:get secrets kind=secret```
    * ```endly kubernetes:apply url=dev-secrets.yaml```
     
## Global contract parameters
- context
- namespace
- kubeconfig
- masterurl
