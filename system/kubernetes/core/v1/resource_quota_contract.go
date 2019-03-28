package v1

import (
	"errors"
	"fmt"
	vvc "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/typed/core/v1"
)

/*autogenerated contract adapter*/

//ResourceQuotaCreateRequest represents request
type ResourceQuotaCreateRequest struct {
	service_ v1.ResourceQuotaInterface
	*vvc.ResourceQuota
}

//ResourceQuotaUpdateRequest represents request
type ResourceQuotaUpdateRequest struct {
	service_ v1.ResourceQuotaInterface
	*vvc.ResourceQuota
}

//ResourceQuotaUpdateStatusRequest represents request
type ResourceQuotaUpdateStatusRequest struct {
	service_ v1.ResourceQuotaInterface
	*vvc.ResourceQuota
}

//ResourceQuotaDeleteRequest represents request
type ResourceQuotaDeleteRequest struct {
	service_ v1.ResourceQuotaInterface
	Name     string
	*metav1.DeleteOptions
}

//ResourceQuotaDeleteCollectionRequest represents request
type ResourceQuotaDeleteCollectionRequest struct {
	service_ v1.ResourceQuotaInterface
	*metav1.DeleteOptions
	ListOptions metav1.ListOptions
}

//ResourceQuotaGetRequest represents request
type ResourceQuotaGetRequest struct {
	service_ v1.ResourceQuotaInterface
	Name     string
	metav1.GetOptions
}

//ResourceQuotaListRequest represents request
type ResourceQuotaListRequest struct {
	service_ v1.ResourceQuotaInterface
	metav1.ListOptions
}

//ResourceQuotaWatchRequest represents request
type ResourceQuotaWatchRequest struct {
	service_ v1.ResourceQuotaInterface
	metav1.ListOptions
}

//ResourceQuotaPatchRequest represents request
type ResourceQuotaPatchRequest struct {
	service_     v1.ResourceQuotaInterface
	Name         string
	Pt           types.PatchType
	Data         []byte
	Subresources []string
}

func init() {
	register(&ResourceQuotaCreateRequest{})
	register(&ResourceQuotaUpdateRequest{})
	register(&ResourceQuotaUpdateStatusRequest{})
	register(&ResourceQuotaDeleteRequest{})
	register(&ResourceQuotaDeleteCollectionRequest{})
	register(&ResourceQuotaGetRequest{})
	register(&ResourceQuotaListRequest{})
	register(&ResourceQuotaWatchRequest{})
	register(&ResourceQuotaPatchRequest{})
}

func (r *ResourceQuotaCreateRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.ResourceQuotaInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.ResourceQuotaInterface", service)
	}
	return nil
}

func (r *ResourceQuotaCreateRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Create(r.ResourceQuota)
	return result, err
}

func (r *ResourceQuotaCreateRequest) GetId() string {
	return "v1.ResourceQuota.Create"
}

func (r *ResourceQuotaUpdateRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.ResourceQuotaInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.ResourceQuotaInterface", service)
	}
	return nil
}

func (r *ResourceQuotaUpdateRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Update(r.ResourceQuota)
	return result, err
}

func (r *ResourceQuotaUpdateRequest) GetId() string {
	return "v1.ResourceQuota.Update"
}

func (r *ResourceQuotaUpdateStatusRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.ResourceQuotaInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.ResourceQuotaInterface", service)
	}
	return nil
}

func (r *ResourceQuotaUpdateStatusRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.UpdateStatus(r.ResourceQuota)
	return result, err
}

func (r *ResourceQuotaUpdateStatusRequest) GetId() string {
	return "v1.ResourceQuota.UpdateStatus"
}

func (r *ResourceQuotaDeleteRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.ResourceQuotaInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.ResourceQuotaInterface", service)
	}
	return nil
}

func (r *ResourceQuotaDeleteRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	err = r.service_.Delete(r.Name, r.DeleteOptions)
	return result, err
}

func (r *ResourceQuotaDeleteRequest) GetId() string {
	return "v1.ResourceQuota.Delete"
}

func (r *ResourceQuotaDeleteCollectionRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.ResourceQuotaInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.ResourceQuotaInterface", service)
	}
	return nil
}

func (r *ResourceQuotaDeleteCollectionRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	err = r.service_.DeleteCollection(r.DeleteOptions, r.ListOptions)
	return result, err
}

func (r *ResourceQuotaDeleteCollectionRequest) GetId() string {
	return "v1.ResourceQuota.DeleteCollection"
}

func (r *ResourceQuotaGetRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.ResourceQuotaInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.ResourceQuotaInterface", service)
	}
	return nil
}

func (r *ResourceQuotaGetRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Get(r.Name, r.GetOptions)
	return result, err
}

func (r *ResourceQuotaGetRequest) GetId() string {
	return "v1.ResourceQuota.Get"
}

func (r *ResourceQuotaListRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.ResourceQuotaInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.ResourceQuotaInterface", service)
	}
	return nil
}

func (r *ResourceQuotaListRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.List(r.ListOptions)
	return result, err
}

func (r *ResourceQuotaListRequest) GetId() string {
	return "v1.ResourceQuota.List"
}

func (r *ResourceQuotaWatchRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.ResourceQuotaInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.ResourceQuotaInterface", service)
	}
	return nil
}

func (r *ResourceQuotaWatchRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Watch(r.ListOptions)
	return result, err
}

func (r *ResourceQuotaWatchRequest) GetId() string {
	return "v1.ResourceQuota.Watch"
}

func (r *ResourceQuotaPatchRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.ResourceQuotaInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.ResourceQuotaInterface", service)
	}
	return nil
}

func (r *ResourceQuotaPatchRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Patch(r.Name, r.Pt, r.Data, r.Subresources...)
	return result, err
}

func (r *ResourceQuotaPatchRequest) GetId() string {
	return "v1.ResourceQuota.Patch"
}
