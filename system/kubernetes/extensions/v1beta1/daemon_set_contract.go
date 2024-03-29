package v1beta1

import (
	"errors"
	"fmt"
	vvc "k8s.io/api/extensions/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
)

/*autogenerated contract adapter*/

// DaemonSetCreateRequest represents request
type DaemonSetCreateRequest struct {
	service_ v1beta1.DaemonSetInterface
	*vvc.DaemonSet
}

// DaemonSetUpdateRequest represents request
type DaemonSetUpdateRequest struct {
	service_ v1beta1.DaemonSetInterface
	*vvc.DaemonSet
}

// DaemonSetUpdateStatusRequest represents request
type DaemonSetUpdateStatusRequest struct {
	service_ v1beta1.DaemonSetInterface
	*vvc.DaemonSet
}

// DaemonSetDeleteRequest represents request
type DaemonSetDeleteRequest struct {
	service_ v1beta1.DaemonSetInterface
	Name     string
	*v1.DeleteOptions
}

// DaemonSetDeleteCollectionRequest represents request
type DaemonSetDeleteCollectionRequest struct {
	service_ v1beta1.DaemonSetInterface
	*v1.DeleteOptions
	ListOptions v1.ListOptions
}

// DaemonSetGetRequest represents request
type DaemonSetGetRequest struct {
	service_ v1beta1.DaemonSetInterface
	Name     string
	v1.GetOptions
}

// DaemonSetListRequest represents request
type DaemonSetListRequest struct {
	service_ v1beta1.DaemonSetInterface
	v1.ListOptions
}

// DaemonSetWatchRequest represents request
type DaemonSetWatchRequest struct {
	service_ v1beta1.DaemonSetInterface
	v1.ListOptions
}

// DaemonSetPatchRequest represents request
type DaemonSetPatchRequest struct {
	service_     v1beta1.DaemonSetInterface
	Name         string
	Pt           types.PatchType
	Data         []byte
	Subresources []string
}

func init() {
	register(&DaemonSetCreateRequest{})
	register(&DaemonSetUpdateRequest{})
	register(&DaemonSetUpdateStatusRequest{})
	register(&DaemonSetDeleteRequest{})
	register(&DaemonSetDeleteCollectionRequest{})
	register(&DaemonSetGetRequest{})
	register(&DaemonSetListRequest{})
	register(&DaemonSetWatchRequest{})
	register(&DaemonSetPatchRequest{})
}

func (r *DaemonSetCreateRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1beta1.DaemonSetInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1beta1.DaemonSetInterface", service)
	}
	return nil
}

func (r *DaemonSetCreateRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Create(r.DaemonSet)
	return result, err
}

func (r *DaemonSetCreateRequest) GetId() string {
	return "extensions/v1beta1.DaemonSet.Create"
}

func (r *DaemonSetUpdateRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1beta1.DaemonSetInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1beta1.DaemonSetInterface", service)
	}
	return nil
}

func (r *DaemonSetUpdateRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Update(r.DaemonSet)
	return result, err
}

func (r *DaemonSetUpdateRequest) GetId() string {
	return "extensions/v1beta1.DaemonSet.Update"
}

func (r *DaemonSetUpdateStatusRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1beta1.DaemonSetInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1beta1.DaemonSetInterface", service)
	}
	return nil
}

func (r *DaemonSetUpdateStatusRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.UpdateStatus(r.DaemonSet)
	return result, err
}

func (r *DaemonSetUpdateStatusRequest) GetId() string {
	return "extensions/v1beta1.DaemonSet.UpdateStatus"
}

func (r *DaemonSetDeleteRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1beta1.DaemonSetInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1beta1.DaemonSetInterface", service)
	}
	return nil
}

func (r *DaemonSetDeleteRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	err = r.service_.Delete(r.Name, r.DeleteOptions)
	return result, err
}

func (r *DaemonSetDeleteRequest) GetId() string {
	return "extensions/v1beta1.DaemonSet.Delete"
}

func (r *DaemonSetDeleteCollectionRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1beta1.DaemonSetInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1beta1.DaemonSetInterface", service)
	}
	return nil
}

func (r *DaemonSetDeleteCollectionRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	err = r.service_.DeleteCollection(r.DeleteOptions, r.ListOptions)
	return result, err
}

func (r *DaemonSetDeleteCollectionRequest) GetId() string {
	return "extensions/v1beta1.DaemonSet.DeleteCollection"
}

func (r *DaemonSetGetRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1beta1.DaemonSetInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1beta1.DaemonSetInterface", service)
	}
	return nil
}

func (r *DaemonSetGetRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Get(r.Name, r.GetOptions)
	return result, err
}

func (r *DaemonSetGetRequest) GetId() string {
	return "extensions/v1beta1.DaemonSet.Get"
}

func (r *DaemonSetListRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1beta1.DaemonSetInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1beta1.DaemonSetInterface", service)
	}
	return nil
}

func (r *DaemonSetListRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.List(r.ListOptions)
	return result, err
}

func (r *DaemonSetListRequest) GetId() string {
	return "extensions/v1beta1.DaemonSet.List"
}

func (r *DaemonSetWatchRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1beta1.DaemonSetInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1beta1.DaemonSetInterface", service)
	}
	return nil
}

func (r *DaemonSetWatchRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Watch(r.ListOptions)
	return result, err
}

func (r *DaemonSetWatchRequest) GetId() string {
	return "extensions/v1beta1.DaemonSet.Watch"
}

func (r *DaemonSetPatchRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1beta1.DaemonSetInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1beta1.DaemonSetInterface", service)
	}
	return nil
}

func (r *DaemonSetPatchRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Patch(r.Name, r.Pt, r.Data, r.Subresources...)
	return result, err
}

func (r *DaemonSetPatchRequest) GetId() string {
	return "extensions/v1beta1.DaemonSet.Patch"
}
