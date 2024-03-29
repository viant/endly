package v1

import (
	"errors"
	"fmt"
	vvc "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/typed/autoscaling/v1"
)

/*autogenerated contract adapter*/

// HorizontalPodAutoscalerCreateRequest represents request
type HorizontalPodAutoscalerCreateRequest struct {
	service_ v1.HorizontalPodAutoscalerInterface
	*vvc.HorizontalPodAutoscaler
}

// HorizontalPodAutoscalerUpdateRequest represents request
type HorizontalPodAutoscalerUpdateRequest struct {
	service_ v1.HorizontalPodAutoscalerInterface
	*vvc.HorizontalPodAutoscaler
}

// HorizontalPodAutoscalerUpdateStatusRequest represents request
type HorizontalPodAutoscalerUpdateStatusRequest struct {
	service_ v1.HorizontalPodAutoscalerInterface
	*vvc.HorizontalPodAutoscaler
}

// HorizontalPodAutoscalerDeleteRequest represents request
type HorizontalPodAutoscalerDeleteRequest struct {
	service_ v1.HorizontalPodAutoscalerInterface
	Name     string
	*metav1.DeleteOptions
}

// HorizontalPodAutoscalerDeleteCollectionRequest represents request
type HorizontalPodAutoscalerDeleteCollectionRequest struct {
	service_ v1.HorizontalPodAutoscalerInterface
	*metav1.DeleteOptions
	ListOptions metav1.ListOptions
}

// HorizontalPodAutoscalerGetRequest represents request
type HorizontalPodAutoscalerGetRequest struct {
	service_ v1.HorizontalPodAutoscalerInterface
	Name     string
	metav1.GetOptions
}

// HorizontalPodAutoscalerListRequest represents request
type HorizontalPodAutoscalerListRequest struct {
	service_ v1.HorizontalPodAutoscalerInterface
	metav1.ListOptions
}

// HorizontalPodAutoscalerWatchRequest represents request
type HorizontalPodAutoscalerWatchRequest struct {
	service_ v1.HorizontalPodAutoscalerInterface
	metav1.ListOptions
}

// HorizontalPodAutoscalerPatchRequest represents request
type HorizontalPodAutoscalerPatchRequest struct {
	service_     v1.HorizontalPodAutoscalerInterface
	Name         string
	Pt           types.PatchType
	Data         []byte
	Subresources []string
}

func init() {
	register(&HorizontalPodAutoscalerCreateRequest{})
	register(&HorizontalPodAutoscalerUpdateRequest{})
	register(&HorizontalPodAutoscalerUpdateStatusRequest{})
	register(&HorizontalPodAutoscalerDeleteRequest{})
	register(&HorizontalPodAutoscalerDeleteCollectionRequest{})
	register(&HorizontalPodAutoscalerGetRequest{})
	register(&HorizontalPodAutoscalerListRequest{})
	register(&HorizontalPodAutoscalerWatchRequest{})
	register(&HorizontalPodAutoscalerPatchRequest{})
}

func (r *HorizontalPodAutoscalerCreateRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.HorizontalPodAutoscalerInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.HorizontalPodAutoscalerInterface", service)
	}
	return nil
}

func (r *HorizontalPodAutoscalerCreateRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Create(r.HorizontalPodAutoscaler)
	return result, err
}

func (r *HorizontalPodAutoscalerCreateRequest) GetId() string {
	return "autoscaling/v1.HorizontalPodAutoscaler.Create"
}

func (r *HorizontalPodAutoscalerUpdateRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.HorizontalPodAutoscalerInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.HorizontalPodAutoscalerInterface", service)
	}
	return nil
}

func (r *HorizontalPodAutoscalerUpdateRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Update(r.HorizontalPodAutoscaler)
	return result, err
}

func (r *HorizontalPodAutoscalerUpdateRequest) GetId() string {
	return "autoscaling/v1.HorizontalPodAutoscaler.Update"
}

func (r *HorizontalPodAutoscalerUpdateStatusRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.HorizontalPodAutoscalerInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.HorizontalPodAutoscalerInterface", service)
	}
	return nil
}

func (r *HorizontalPodAutoscalerUpdateStatusRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.UpdateStatus(r.HorizontalPodAutoscaler)
	return result, err
}

func (r *HorizontalPodAutoscalerUpdateStatusRequest) GetId() string {
	return "autoscaling/v1.HorizontalPodAutoscaler.UpdateStatus"
}

func (r *HorizontalPodAutoscalerDeleteRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.HorizontalPodAutoscalerInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.HorizontalPodAutoscalerInterface", service)
	}
	return nil
}

func (r *HorizontalPodAutoscalerDeleteRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	err = r.service_.Delete(r.Name, r.DeleteOptions)
	return result, err
}

func (r *HorizontalPodAutoscalerDeleteRequest) GetId() string {
	return "autoscaling/v1.HorizontalPodAutoscaler.Delete"
}

func (r *HorizontalPodAutoscalerDeleteCollectionRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.HorizontalPodAutoscalerInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.HorizontalPodAutoscalerInterface", service)
	}
	return nil
}

func (r *HorizontalPodAutoscalerDeleteCollectionRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	err = r.service_.DeleteCollection(r.DeleteOptions, r.ListOptions)
	return result, err
}

func (r *HorizontalPodAutoscalerDeleteCollectionRequest) GetId() string {
	return "autoscaling/v1.HorizontalPodAutoscaler.DeleteCollection"
}

func (r *HorizontalPodAutoscalerGetRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.HorizontalPodAutoscalerInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.HorizontalPodAutoscalerInterface", service)
	}
	return nil
}

func (r *HorizontalPodAutoscalerGetRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Get(r.Name, r.GetOptions)
	return result, err
}

func (r *HorizontalPodAutoscalerGetRequest) GetId() string {
	return "autoscaling/v1.HorizontalPodAutoscaler.Get"
}

func (r *HorizontalPodAutoscalerListRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.HorizontalPodAutoscalerInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.HorizontalPodAutoscalerInterface", service)
	}
	return nil
}

func (r *HorizontalPodAutoscalerListRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.List(r.ListOptions)
	return result, err
}

func (r *HorizontalPodAutoscalerListRequest) GetId() string {
	return "autoscaling/v1.HorizontalPodAutoscaler.List"
}

func (r *HorizontalPodAutoscalerWatchRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.HorizontalPodAutoscalerInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.HorizontalPodAutoscalerInterface", service)
	}
	return nil
}

func (r *HorizontalPodAutoscalerWatchRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Watch(r.ListOptions)
	return result, err
}

func (r *HorizontalPodAutoscalerWatchRequest) GetId() string {
	return "autoscaling/v1.HorizontalPodAutoscaler.Watch"
}

func (r *HorizontalPodAutoscalerPatchRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.HorizontalPodAutoscalerInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.HorizontalPodAutoscalerInterface", service)
	}
	return nil
}

func (r *HorizontalPodAutoscalerPatchRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Patch(r.Name, r.Pt, r.Data, r.Subresources...)
	return result, err
}

func (r *HorizontalPodAutoscalerPatchRequest) GetId() string {
	return "autoscaling/v1.HorizontalPodAutoscaler.Patch"
}
