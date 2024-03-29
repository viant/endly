package v1

import (
	"errors"
	"fmt"
	vvc "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/typed/rbac/v1"
)

/*autogenerated contract adapter*/

// RoleBindingCreateRequest represents request
type RoleBindingCreateRequest struct {
	service_ v1.RoleBindingInterface
	*vvc.RoleBinding
}

// RoleBindingUpdateRequest represents request
type RoleBindingUpdateRequest struct {
	service_ v1.RoleBindingInterface
	*vvc.RoleBinding
}

// RoleBindingDeleteRequest represents request
type RoleBindingDeleteRequest struct {
	service_ v1.RoleBindingInterface
	Name     string
	*metav1.DeleteOptions
}

// RoleBindingDeleteCollectionRequest represents request
type RoleBindingDeleteCollectionRequest struct {
	service_ v1.RoleBindingInterface
	*metav1.DeleteOptions
	ListOptions metav1.ListOptions
}

// RoleBindingGetRequest represents request
type RoleBindingGetRequest struct {
	service_ v1.RoleBindingInterface
	Name     string
	metav1.GetOptions
}

// RoleBindingListRequest represents request
type RoleBindingListRequest struct {
	service_ v1.RoleBindingInterface
	metav1.ListOptions
}

// RoleBindingWatchRequest represents request
type RoleBindingWatchRequest struct {
	service_ v1.RoleBindingInterface
	metav1.ListOptions
}

// RoleBindingPatchRequest represents request
type RoleBindingPatchRequest struct {
	service_     v1.RoleBindingInterface
	Name         string
	Pt           types.PatchType
	Data         []byte
	Subresources []string
}

func init() {
	register(&RoleBindingCreateRequest{})
	register(&RoleBindingUpdateRequest{})
	register(&RoleBindingDeleteRequest{})
	register(&RoleBindingDeleteCollectionRequest{})
	register(&RoleBindingGetRequest{})
	register(&RoleBindingListRequest{})
	register(&RoleBindingWatchRequest{})
	register(&RoleBindingPatchRequest{})
}

func (r *RoleBindingCreateRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.RoleBindingInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.RoleBindingInterface", service)
	}
	return nil
}

func (r *RoleBindingCreateRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Create(r.RoleBinding)
	return result, err
}

func (r *RoleBindingCreateRequest) GetId() string {
	return "rbac/v1.RoleBinding.Create"
}

func (r *RoleBindingUpdateRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.RoleBindingInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.RoleBindingInterface", service)
	}
	return nil
}

func (r *RoleBindingUpdateRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Update(r.RoleBinding)
	return result, err
}

func (r *RoleBindingUpdateRequest) GetId() string {
	return "rbac/v1.RoleBinding.Update"
}

func (r *RoleBindingDeleteRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.RoleBindingInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.RoleBindingInterface", service)
	}
	return nil
}

func (r *RoleBindingDeleteRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	err = r.service_.Delete(r.Name, r.DeleteOptions)
	return result, err
}

func (r *RoleBindingDeleteRequest) GetId() string {
	return "rbac/v1.RoleBinding.Delete"
}

func (r *RoleBindingDeleteCollectionRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.RoleBindingInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.RoleBindingInterface", service)
	}
	return nil
}

func (r *RoleBindingDeleteCollectionRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	err = r.service_.DeleteCollection(r.DeleteOptions, r.ListOptions)
	return result, err
}

func (r *RoleBindingDeleteCollectionRequest) GetId() string {
	return "rbac/v1.RoleBinding.DeleteCollection"
}

func (r *RoleBindingGetRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.RoleBindingInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.RoleBindingInterface", service)
	}
	return nil
}

func (r *RoleBindingGetRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Get(r.Name, r.GetOptions)
	return result, err
}

func (r *RoleBindingGetRequest) GetId() string {
	return "rbac/v1.RoleBinding.Get"
}

func (r *RoleBindingListRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.RoleBindingInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.RoleBindingInterface", service)
	}
	return nil
}

func (r *RoleBindingListRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.List(r.ListOptions)
	return result, err
}

func (r *RoleBindingListRequest) GetId() string {
	return "rbac/v1.RoleBinding.List"
}

func (r *RoleBindingWatchRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.RoleBindingInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.RoleBindingInterface", service)
	}
	return nil
}

func (r *RoleBindingWatchRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Watch(r.ListOptions)
	return result, err
}

func (r *RoleBindingWatchRequest) GetId() string {
	return "rbac/v1.RoleBinding.Watch"
}

func (r *RoleBindingPatchRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.RoleBindingInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.RoleBindingInterface", service)
	}
	return nil
}

func (r *RoleBindingPatchRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Patch(r.Name, r.Pt, r.Data, r.Subresources...)
	return result, err
}

func (r *RoleBindingPatchRequest) GetId() string {
	return "rbac/v1.RoleBinding.Patch"
}
