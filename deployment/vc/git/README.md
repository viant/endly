## Version Control/Git Service

This service uses gopkg.in/src-d/go-git.v4 git client


| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| version/control | status | run version control check on provided URL | [StatusRequest](serivce_contract.go) | [Info](serivce_contract.go)  |
| version/control | checkout | if target directory already  exist with matching origin URL, this action only pulls the latest changes without overriding local ones, otherwise full checkout | [CheckoutRequest](serivce_contract.go) | [Info](serivce_contract.go) |


### Usage


### Checkout/pull


1. Basic checkout
    * ```endly -r=checkout```
    * [@checkout.yaml](checkout.yaml)
    ```yaml
    pipeline:
      checkout:
        action: vc/git:checkout
        origin:
          URL: https://github.com/src-d/go-git
        dest:
          URL: /tmp/foo
    ```
2. Private repo with basic auth
    * endly -c=myacount
    * ```endly -r=checkout```
    * [@checkout.yaml](checkout_private.yaml)
    ```yaml
    pipeline:
      checkout:
        action: vc/git:checkout
        origin:
          URL: https://github.com/myacount/myrepo.git
          credentials: myacount
        dest:
          URL: /tmp/myrepo
    ```
3. Private key based auth
    * ```endly -c=myacountpk -k=/path/key_rsa```  where username is git and password is pharaprase  
    * ```endly -r=checkout``` 
    * [@checkout.yaml](checkout_pk.yaml)
    ```yaml
    pipeline:
      checkout:
        action: vc/git:checkout
        origin:
          URL: git@github.com/myacount/myrepo.git
          credentials: myrepopk
        dest:
          URL: /tmp/myrepo
    ```
