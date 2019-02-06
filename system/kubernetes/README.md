## Kubernetes service

This service is *k8s.io/client-go/kubernetes.Clientset proxy 

It defines helper method to deal with basic operation with human friendly way.


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




work in progress


## Global contract parameters

- context
- namespace
- 