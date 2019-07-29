# Google Cloud Key Management Service

This service is google.golang.org/api/cloudkms/v1/Service proxy 

To check all supported method run
```bash
     endly -s='gcp/kms'
```

To check method contract run endly -s='gcp/kms' -a=methodName
```bash
    endly -s='gcp/kms:keyRingsList' 
```

_References:_
- [KMS API](https://cloud.google.com/kms/docs/reference/rest/)


#### Usage:

##### Creating a symmetric key

- **With gcloud**
```bash
gcloud kms keyrings create my_ring --location us-central1

gcloud kms keys create my_key --location us-central1 \
  --keyring my_ring --purpose encryption

```

- **With endly**

```bash
endly deploy
```

[@deploy.yaml](usage/deploy.yaml)

```yaml
pipeline:
  secure:
    deployKey:
      action: gcp/kms:deployKey
      credentials: gcp-e2e
      ring: my_ring
      key: my_key
      purpose: ENCRYPT_DECRYPT
      bindings:
        - role: roles/cloudkms.cryptoKeyEncrypterDecrypter
          members:
            - user:awitas@vindicotech.com
            - serviceAccount:${gcp.serviceAccount}

    keyInfo:
      action: print
      message: 'Deployed key: ${deployKey.Primary.Name}'

```

##### Encryption with a symmetric key 


- **With gcloud**

```bash
gcloud kms encrypt \
  --location=us-central1  \
  --keyring=my_ring \
  --key=my_key \
  --version=1 \
  --plaintext-file=data.txt \
  --ciphertext-file=data.enc
```


- **With endly**


```bash
endly decrypt
```

[@encrypt.yaml](usage/encrypt.yaml)
```yaml
pipeline:
  encrypt:
    action: gcp/kms:encrypt
    logging: false
    ring: my_ring
    key: my_key
    source:
      URL: data.txt
    dest:
      URL: data.enc
```

##### Decryption with a symmetric key 


- **With gcloud**


```bash

gcloud kms decrypt \
  --location=us-central1 \
  --keyring=my_ring \
  --key=my_key \
  --ciphertext-file=data.enc \
  --plaintext-file=data.dec
```


- **With endly**


```bash
endly decrypt
```

[@encrypt.yaml](usage/decrypt.yaml)

```yaml
pipeline:
  encrypt:
    action: gcp/kms:encrypt
    logging: false
    ring: my_ring
    key: my_key
    source:
      URL: data.txt
    dest:
      URL: data.enc
```



##### Inline encryption/decryption

```bash
endy -r=inline
```

[@inline.yaml](usage/inline.yaml)
```yaml
pipeline:
  secure:
    deployKey:
      action: gcp/kms:deployKey
      credentials: gcp-e2e
      ring: my_ring
      key: my_key
      purpose: ENCRYPT_DECRYPT
      logging: false
      bindings:
        - role: roles/cloudkms.cryptoKeyEncrypterDecrypter
          members:
            - user:awitas@vindicotech.com
            - serviceAccount:${gcp.serviceAccount}

    keyInfo:
      action: print
      message: 'Deployed key: ${deployKey.Primary.Name}'

    encrypt:
      action: gcp/kms:encrypt
      ring: my_ring
      key: my_key
      plainData: this is test
      logging: false
      
    decrypt:
      action: gcp/kms:decrypt
      ring: my_ring
      key: my_key
      cipherBase64Text: ${encrypt.CipherBase64Text}
      logging: false
    info:
      action: print
      message: 'decrypted:  $AsString(${decrypt.PlainData})'
```



##### Google storage asset encryption/decryption


```bash
endy -r=secure
```

[@secure.yaml](usage/secure.yaml)
```yaml
pipeline:
  secure:
    deployKey:
      action: gcp/kms:deployKey
      credentials: gcp-e2e
      ring: my_ring
      key: my_key
      logging: false
      purpose: ENCRYPT_DECRYPT
      bindings:
        - role: roles/cloudkms.cryptoKeyEncrypterDecrypter
          members:
            - serviceAccount:$gcp.serviceAccount

    encrypt:
      action: gcp/kms:encrypt
      logging: false
      ring: my_ring
      key: my_key
      plainData: this is test
      dest:
        URL: /tmp/config.json.enc
    decrypt:
      action: gcp/kms:decrypt
      logging: false
      ring: my_ring
      key: my_key
      source:
        URL: /tmp/config.json.enc
    info:
      action: print
      message: $AsString(${decrypt.PlainData})

```


##### Accessing encrypted URL asset 
 
```go
package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"google.golang.org/api/cloudkms/v1"
	"google.golang.org/api/option"
	"log"
	_ "github.com/viant/toolbox/storage/gs"
	"github.com/viant/toolbox/url"
	"os"
	"path"
)

func main() {

	resource := url.NewResource("gs://myBucket/config.json.enc")
	keyURI := "projects/MY_PROJECT/locations/REGION/keyRings/my_ring/cryptoKeys/my_key"
	plain, err := decrypt(keyURI, resource)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", plain)
}

func decrypt(key string, resource *url.Resource) ([]byte, error) {
	data, err := resource.DownloadText()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	kmsService, err := cloudkms.NewService(ctx, option.WithScopes(cloudkms.CloudPlatformScope, cloudkms.CloudkmsScope))
	if err != nil {
		return nil, err
	}
	service := cloudkms.NewProjectsLocationsKeyRingsCryptoKeysService(kmsService)
	response, err := service.Decrypt(key, &cloudkms.DecryptRequest{Ciphertext:data}).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(string(response.Plaintext))
}

``` 