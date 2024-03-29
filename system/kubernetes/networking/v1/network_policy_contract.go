package v1

import (
	"errors"
	"fmt"
	vvc "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/typed/networking/v1"
)

/*autogenerated contract adapter*/

// NetworkPolicyCreateRequest represents request
type NetworkPolicyCreateRequest struct {
	service_ v1.NetworkPolicyInterface
	*vvc.NetworkPolicy
}

// NetworkPolicyUpdateRequest represents request
type NetworkPolicyUpdateRequest struct {
	service_ v1.NetworkPolicyInterface
	*vvc.NetworkPolicy
}

// NetworkPolicyDeleteRequest represents request
type NetworkPolicyDeleteRequest struct {
	service_ v1.NetworkPolicyInterface
	Name     string
	*metav1.DeleteOptions
}

// NetworkPolicyDeleteCollectionRequest represents request
type NetworkPolicyDeleteCollectionRequest struct {
	service_ v1.NetworkPolicyInterface
	*metav1.DeleteOptions
	ListOptions metav1.ListOptions
}

// NetworkPolicyGetRequest represents request
type NetworkPolicyGetRequest struct {
	service_ v1.NetworkPolicyInterface
	Name     string
	metav1.GetOptions
}

// NetworkPolicyListRequest represents request
type NetworkPolicyListRequest struct {
	service_ v1.NetworkPolicyInterface
	metav1.ListOptions
}

// NetworkPolicyWatchRequest represents request
type NetworkPolicyWatchRequest struct {
	service_ v1.NetworkPolicyInterface
	metav1.ListOptions
}

// NetworkPolicyPatchRequest represents request
type NetworkPolicyPatchRequest struct {
	service_     v1.NetworkPolicyInterface
	Name         string
	Pt           types.PatchType
	Data         []byte
	Subresources []string
}

func init() {
	register(&NetworkPolicyCreateRequest{})
	register(&NetworkPolicyUpdateRequest{})
	register(&NetworkPolicyDeleteRequest{})
	register(&NetworkPolicyDeleteCollectionRequest{})
	register(&NetworkPolicyGetRequest{})
	register(&NetworkPolicyListRequest{})
	register(&NetworkPolicyWatchRequest{})
	register(&NetworkPolicyPatchRequest{})
}

func (r *NetworkPolicyCreateRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.NetworkPolicyInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.NetworkPolicyInterface", service)
	}
	return nil
}

func (r *NetworkPolicyCreateRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Create(r.NetworkPolicy)
	return result, err
}

func (r *NetworkPolicyCreateRequest) GetId() string {
	return "networking/v1.NetworkPolicy.Create"
}

func (r *NetworkPolicyUpdateRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.NetworkPolicyInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.NetworkPolicyInterface", service)
	}
	return nil
}

func (r *NetworkPolicyUpdateRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Update(r.NetworkPolicy)
	return result, err
}

func (r *NetworkPolicyUpdateRequest) GetId() string {
	return "networking/v1.NetworkPolicy.Update"
}

func (r *NetworkPolicyDeleteRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.NetworkPolicyInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.NetworkPolicyInterface", service)
	}
	return nil
}

func (r *NetworkPolicyDeleteRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	err = r.service_.Delete(r.Name, r.DeleteOptions)
	return result, err
}

func (r *NetworkPolicyDeleteRequest) GetId() string {
	return "networking/v1.NetworkPolicy.Delete"
}

func (r *NetworkPolicyDeleteCollectionRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.NetworkPolicyInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.NetworkPolicyInterface", service)
	}
	return nil
}

func (r *NetworkPolicyDeleteCollectionRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	err = r.service_.DeleteCollection(r.DeleteOptions, r.ListOptions)
	return result, err
}

func (r *NetworkPolicyDeleteCollectionRequest) GetId() string {
	return "networking/v1.NetworkPolicy.DeleteCollection"
}

func (r *NetworkPolicyGetRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.NetworkPolicyInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.NetworkPolicyInterface", service)
	}
	return nil
}

func (r *NetworkPolicyGetRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Get(r.Name, r.GetOptions)
	return result, err
}

func (r *NetworkPolicyGetRequest) GetId() string {
	return "networking/v1.NetworkPolicy.Get"
}

func (r *NetworkPolicyListRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.NetworkPolicyInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.NetworkPolicyInterface", service)
	}
	return nil
}

func (r *NetworkPolicyListRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.List(r.ListOptions)
	return result, err
}

func (r *NetworkPolicyListRequest) GetId() string {
	return "networking/v1.NetworkPolicy.List"
}

func (r *NetworkPolicyWatchRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.NetworkPolicyInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.NetworkPolicyInterface", service)
	}
	return nil
}

func (r *NetworkPolicyWatchRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Watch(r.ListOptions)
	return result, err
}

func (r *NetworkPolicyWatchRequest) GetId() string {
	return "networking/v1.NetworkPolicy.Watch"
}

func (r *NetworkPolicyPatchRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.NetworkPolicyInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.NetworkPolicyInterface", service)
	}
	return nil
}

func (r *NetworkPolicyPatchRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Patch(r.Name, r.Pt, r.Data, r.Subresources...)
	return result, err
}

func (r *NetworkPolicyPatchRequest) GetId() string {
	return "networking/v1.NetworkPolicy.Patch"
}
