package v1



import (
	"fmt"
"errors"
	"k8s.io/client-go/kubernetes/typed/core/v1"
	types "k8s.io/apimachinery/pkg/types"
	vvc "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"	
)

/*autogenerated contract adapter*/

//SecretCreateRequest represents request
type SecretCreateRequest struct {
  service_ v1.SecretInterface
   *vvc.Secret
}

//SecretUpdateRequest represents request
type SecretUpdateRequest struct {
  service_ v1.SecretInterface
   *vvc.Secret
}

//SecretDeleteRequest represents request
type SecretDeleteRequest struct {
  service_ v1.SecretInterface
  Name string
   *metav1.DeleteOptions
}

//SecretDeleteCollectionRequest represents request
type SecretDeleteCollectionRequest struct {
  service_ v1.SecretInterface
   *metav1.DeleteOptions
  ListOptions metav1.ListOptions
}

//SecretGetRequest represents request
type SecretGetRequest struct {
  service_ v1.SecretInterface
  Name string
   metav1.GetOptions
}

//SecretListRequest represents request
type SecretListRequest struct {
  service_ v1.SecretInterface
   metav1.ListOptions
}

//SecretWatchRequest represents request
type SecretWatchRequest struct {
  service_ v1.SecretInterface
   metav1.ListOptions
}

//SecretPatchRequest represents request
type SecretPatchRequest struct {
  service_ v1.SecretInterface
  Name string
  Pt types.PatchType
  Data []byte
  Subresources []string
}


func init() {
	register(&SecretCreateRequest{})
	register(&SecretUpdateRequest{})
	register(&SecretDeleteRequest{})
	register(&SecretDeleteCollectionRequest{})
	register(&SecretGetRequest{})
	register(&SecretListRequest{})
	register(&SecretWatchRequest{})
	register(&SecretPatchRequest{})
}


func (r * SecretCreateRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.SecretInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.SecretInterface", service)
	}
	return nil
}

func (r * SecretCreateRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Create(r.Secret)
	return result, err	
}

func (r * SecretCreateRequest) GetId() string {
	return "v1.Secret.Create";	
}

func (r * SecretUpdateRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.SecretInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.SecretInterface", service)
	}
	return nil
}

func (r * SecretUpdateRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Update(r.Secret)
	return result, err	
}

func (r * SecretUpdateRequest) GetId() string {
	return "v1.Secret.Update";	
}

func (r * SecretDeleteRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.SecretInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.SecretInterface", service)
	}
	return nil
}

func (r * SecretDeleteRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	err = r.service_.Delete(r.Name,r.DeleteOptions)
	return result, err	
}

func (r * SecretDeleteRequest) GetId() string {
	return "v1.Secret.Delete";	
}

func (r * SecretDeleteCollectionRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.SecretInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.SecretInterface", service)
	}
	return nil
}

func (r * SecretDeleteCollectionRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	err = r.service_.DeleteCollection(r.DeleteOptions,r.ListOptions)
	return result, err	
}

func (r * SecretDeleteCollectionRequest) GetId() string {
	return "v1.Secret.DeleteCollection";	
}

func (r * SecretGetRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.SecretInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.SecretInterface", service)
	}
	return nil
}

func (r * SecretGetRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Get(r.Name,r.GetOptions)
	return result, err	
}

func (r * SecretGetRequest) GetId() string {
	return "v1.Secret.Get";	
}

func (r * SecretListRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.SecretInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.SecretInterface", service)
	}
	return nil
}

func (r * SecretListRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.List(r.ListOptions)
	return result, err	
}

func (r * SecretListRequest) GetId() string {
	return "v1.Secret.List";	
}

func (r * SecretWatchRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.SecretInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.SecretInterface", service)
	}
	return nil
}

func (r * SecretWatchRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Watch(r.ListOptions)
	return result, err	
}

func (r * SecretWatchRequest) GetId() string {
	return "v1.Secret.Watch";	
}

func (r * SecretPatchRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.SecretInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.SecretInterface", service)
	}
	return nil
}

func (r * SecretPatchRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Patch(r.Name,r.Pt,r.Data,r.Subresources...)
	return result, err	
}

func (r * SecretPatchRequest) GetId() string {
	return "v1.Secret.Patch";	
}