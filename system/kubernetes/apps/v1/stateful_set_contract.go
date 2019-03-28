package v1

import (
	"errors"
	"fmt"
	vvc "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/typed/apps/v1"
)

/*autogenerated contract adapter*/

//StatefulSetCreateRequest represents request
type StatefulSetCreateRequest struct {
	service_ v1.StatefulSetInterface
	*vvc.StatefulSet
}

//StatefulSetUpdateRequest represents request
type StatefulSetUpdateRequest struct {
	service_ v1.StatefulSetInterface
	*vvc.StatefulSet
}

//StatefulSetUpdateStatusRequest represents request
type StatefulSetUpdateStatusRequest struct {
	service_ v1.StatefulSetInterface
	*vvc.StatefulSet
}

//StatefulSetDeleteRequest represents request
type StatefulSetDeleteRequest struct {
	service_ v1.StatefulSetInterface
	Name     string
	*metav1.DeleteOptions
}

//StatefulSetDeleteCollectionRequest represents request
type StatefulSetDeleteCollectionRequest struct {
	service_ v1.StatefulSetInterface
	*metav1.DeleteOptions
	ListOptions metav1.ListOptions
}

//StatefulSetGetRequest represents request
type StatefulSetGetRequest struct {
	service_ v1.StatefulSetInterface
	Name     string
	metav1.GetOptions
}

//StatefulSetListRequest represents request
type StatefulSetListRequest struct {
	service_ v1.StatefulSetInterface
	metav1.ListOptions
}

//StatefulSetWatchRequest represents request
type StatefulSetWatchRequest struct {
	service_ v1.StatefulSetInterface
	metav1.ListOptions
}

//StatefulSetPatchRequest represents request
type StatefulSetPatchRequest struct {
	service_     v1.StatefulSetInterface
	Name         string
	Pt           types.PatchType
	Data         []byte
	Subresources []string
}

//StatefulSetGetScaleRequest represents request
type StatefulSetGetScaleRequest struct {
	service_        v1.StatefulSetInterface
	StatefulSetName string
	metav1.GetOptions
}

//StatefulSetUpdateScaleRequest represents request
type StatefulSetUpdateScaleRequest struct {
	service_        v1.StatefulSetInterface
	StatefulSetName string
	*autoscalingv1.Scale
}

func init() {
	register(&StatefulSetCreateRequest{})
	register(&StatefulSetUpdateRequest{})
	register(&StatefulSetUpdateStatusRequest{})
	register(&StatefulSetDeleteRequest{})
	register(&StatefulSetDeleteCollectionRequest{})
	register(&StatefulSetGetRequest{})
	register(&StatefulSetListRequest{})
	register(&StatefulSetWatchRequest{})
	register(&StatefulSetPatchRequest{})
	register(&StatefulSetGetScaleRequest{})
	register(&StatefulSetUpdateScaleRequest{})
}

func (r *StatefulSetCreateRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.StatefulSetInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.StatefulSetInterface", service)
	}
	return nil
}

func (r *StatefulSetCreateRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Create(r.StatefulSet)
	return result, err
}

func (r *StatefulSetCreateRequest) GetId() string {
	return "apps/v1.StatefulSet.Create"
}

func (r *StatefulSetUpdateRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.StatefulSetInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.StatefulSetInterface", service)
	}
	return nil
}

func (r *StatefulSetUpdateRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Update(r.StatefulSet)
	return result, err
}

func (r *StatefulSetUpdateRequest) GetId() string {
	return "apps/v1.StatefulSet.Update"
}

func (r *StatefulSetUpdateStatusRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.StatefulSetInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.StatefulSetInterface", service)
	}
	return nil
}

func (r *StatefulSetUpdateStatusRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.UpdateStatus(r.StatefulSet)
	return result, err
}

func (r *StatefulSetUpdateStatusRequest) GetId() string {
	return "apps/v1.StatefulSet.UpdateStatus"
}

func (r *StatefulSetDeleteRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.StatefulSetInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.StatefulSetInterface", service)
	}
	return nil
}

func (r *StatefulSetDeleteRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	err = r.service_.Delete(r.Name, r.DeleteOptions)
	return result, err
}

func (r *StatefulSetDeleteRequest) GetId() string {
	return "apps/v1.StatefulSet.Delete"
}

func (r *StatefulSetDeleteCollectionRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.StatefulSetInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.StatefulSetInterface", service)
	}
	return nil
}

func (r *StatefulSetDeleteCollectionRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	err = r.service_.DeleteCollection(r.DeleteOptions, r.ListOptions)
	return result, err
}

func (r *StatefulSetDeleteCollectionRequest) GetId() string {
	return "apps/v1.StatefulSet.DeleteCollection"
}

func (r *StatefulSetGetRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.StatefulSetInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.StatefulSetInterface", service)
	}
	return nil
}

func (r *StatefulSetGetRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Get(r.Name, r.GetOptions)
	return result, err
}

func (r *StatefulSetGetRequest) GetId() string {
	return "apps/v1.StatefulSet.Get"
}

func (r *StatefulSetListRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.StatefulSetInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.StatefulSetInterface", service)
	}
	return nil
}

func (r *StatefulSetListRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.List(r.ListOptions)
	return result, err
}

func (r *StatefulSetListRequest) GetId() string {
	return "apps/v1.StatefulSet.List"
}

func (r *StatefulSetWatchRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.StatefulSetInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.StatefulSetInterface", service)
	}
	return nil
}

func (r *StatefulSetWatchRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Watch(r.ListOptions)
	return result, err
}

func (r *StatefulSetWatchRequest) GetId() string {
	return "apps/v1.StatefulSet.Watch"
}

func (r *StatefulSetPatchRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.StatefulSetInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.StatefulSetInterface", service)
	}
	return nil
}

func (r *StatefulSetPatchRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Patch(r.Name, r.Pt, r.Data, r.Subresources...)
	return result, err
}

func (r *StatefulSetPatchRequest) GetId() string {
	return "apps/v1.StatefulSet.Patch"
}

func (r *StatefulSetGetScaleRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.StatefulSetInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.StatefulSetInterface", service)
	}
	return nil
}

func (r *StatefulSetGetScaleRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.GetScale(r.StatefulSetName, r.GetOptions)
	return result, err
}

func (r *StatefulSetGetScaleRequest) GetId() string {
	return "apps/v1.StatefulSet.GetScale"
}

func (r *StatefulSetUpdateScaleRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.StatefulSetInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.StatefulSetInterface", service)
	}
	return nil
}

func (r *StatefulSetUpdateScaleRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.UpdateScale(r.StatefulSetName, r.Scale)
	return result, err
}

func (r *StatefulSetUpdateScaleRequest) GetId() string {
	return "apps/v1.StatefulSet.UpdateScale"
}
