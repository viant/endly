package v1



import (
	"fmt"
"errors"
	"k8s.io/client-go/kubernetes/typed/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	vvc "k8s.io/api/apps/v1"	
)

/*autogenerated contract adapter*/

//ControllerRevisionCreateRequest represents request
type ControllerRevisionCreateRequest struct {
  service_ v1.ControllerRevisionInterface
   *vvc.ControllerRevision
}

//ControllerRevisionUpdateRequest represents request
type ControllerRevisionUpdateRequest struct {
  service_ v1.ControllerRevisionInterface
   *vvc.ControllerRevision
}

//ControllerRevisionDeleteRequest represents request
type ControllerRevisionDeleteRequest struct {
  service_ v1.ControllerRevisionInterface
  Name string
   *metav1.DeleteOptions
}

//ControllerRevisionDeleteCollectionRequest represents request
type ControllerRevisionDeleteCollectionRequest struct {
  service_ v1.ControllerRevisionInterface
   *metav1.DeleteOptions
  ListOptions metav1.ListOptions
}

//ControllerRevisionGetRequest represents request
type ControllerRevisionGetRequest struct {
  service_ v1.ControllerRevisionInterface
  Name string
   metav1.GetOptions
}

//ControllerRevisionListRequest represents request
type ControllerRevisionListRequest struct {
  service_ v1.ControllerRevisionInterface
   metav1.ListOptions
}

//ControllerRevisionWatchRequest represents request
type ControllerRevisionWatchRequest struct {
  service_ v1.ControllerRevisionInterface
   metav1.ListOptions
}

//ControllerRevisionPatchRequest represents request
type ControllerRevisionPatchRequest struct {
  service_ v1.ControllerRevisionInterface
  Name string
  Pt types.PatchType
  Data []byte
  Subresources []string
}


func init() {
	register(&ControllerRevisionCreateRequest{})
	register(&ControllerRevisionUpdateRequest{})
	register(&ControllerRevisionDeleteRequest{})
	register(&ControllerRevisionDeleteCollectionRequest{})
	register(&ControllerRevisionGetRequest{})
	register(&ControllerRevisionListRequest{})
	register(&ControllerRevisionWatchRequest{})
	register(&ControllerRevisionPatchRequest{})
}


func (r * ControllerRevisionCreateRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.ControllerRevisionInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.ControllerRevisionInterface", service)
	}
	return nil
}

func (r * ControllerRevisionCreateRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Create(r.ControllerRevision)
	return result, err	
}

func (r * ControllerRevisionCreateRequest) GetId() string {
	return "apps/v1.ControllerRevision.Create";	
}

func (r * ControllerRevisionUpdateRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.ControllerRevisionInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.ControllerRevisionInterface", service)
	}
	return nil
}

func (r * ControllerRevisionUpdateRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Update(r.ControllerRevision)
	return result, err	
}

func (r * ControllerRevisionUpdateRequest) GetId() string {
	return "apps/v1.ControllerRevision.Update";	
}

func (r * ControllerRevisionDeleteRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.ControllerRevisionInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.ControllerRevisionInterface", service)
	}
	return nil
}

func (r * ControllerRevisionDeleteRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	err = r.service_.Delete(r.Name,r.DeleteOptions)
	return result, err	
}

func (r * ControllerRevisionDeleteRequest) GetId() string {
	return "apps/v1.ControllerRevision.Delete";	
}

func (r * ControllerRevisionDeleteCollectionRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.ControllerRevisionInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.ControllerRevisionInterface", service)
	}
	return nil
}

func (r * ControllerRevisionDeleteCollectionRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	err = r.service_.DeleteCollection(r.DeleteOptions,r.ListOptions)
	return result, err	
}

func (r * ControllerRevisionDeleteCollectionRequest) GetId() string {
	return "apps/v1.ControllerRevision.DeleteCollection";	
}

func (r * ControllerRevisionGetRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.ControllerRevisionInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.ControllerRevisionInterface", service)
	}
	return nil
}

func (r * ControllerRevisionGetRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Get(r.Name,r.GetOptions)
	return result, err	
}

func (r * ControllerRevisionGetRequest) GetId() string {
	return "apps/v1.ControllerRevision.Get";	
}

func (r * ControllerRevisionListRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.ControllerRevisionInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.ControllerRevisionInterface", service)
	}
	return nil
}

func (r * ControllerRevisionListRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.List(r.ListOptions)
	return result, err	
}

func (r * ControllerRevisionListRequest) GetId() string {
	return "apps/v1.ControllerRevision.List";	
}

func (r * ControllerRevisionWatchRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.ControllerRevisionInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.ControllerRevisionInterface", service)
	}
	return nil
}

func (r * ControllerRevisionWatchRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Watch(r.ListOptions)
	return result, err	
}

func (r * ControllerRevisionWatchRequest) GetId() string {
	return "apps/v1.ControllerRevision.Watch";	
}

func (r * ControllerRevisionPatchRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.ControllerRevisionInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.ControllerRevisionInterface", service)
	}
	return nil
}

func (r * ControllerRevisionPatchRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Patch(r.Name,r.Pt,r.Data,r.Subresources...)
	return result, err	
}

func (r * ControllerRevisionPatchRequest) GetId() string {
	return "apps/v1.ControllerRevision.Patch";	
}