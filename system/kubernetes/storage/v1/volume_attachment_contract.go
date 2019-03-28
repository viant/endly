package v1

import (
	"errors"
	"fmt"
	vvc "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/typed/storage/v1"
)

/*autogenerated contract adapter*/

//VolumeAttachmentCreateRequest represents request
type VolumeAttachmentCreateRequest struct {
	service_ v1.VolumeAttachmentInterface
	*vvc.VolumeAttachment
}

//VolumeAttachmentUpdateRequest represents request
type VolumeAttachmentUpdateRequest struct {
	service_ v1.VolumeAttachmentInterface
	*vvc.VolumeAttachment
}

//VolumeAttachmentUpdateStatusRequest represents request
type VolumeAttachmentUpdateStatusRequest struct {
	service_ v1.VolumeAttachmentInterface
	*vvc.VolumeAttachment
}

//VolumeAttachmentDeleteRequest represents request
type VolumeAttachmentDeleteRequest struct {
	service_ v1.VolumeAttachmentInterface
	Name     string
	*metav1.DeleteOptions
}

//VolumeAttachmentDeleteCollectionRequest represents request
type VolumeAttachmentDeleteCollectionRequest struct {
	service_ v1.VolumeAttachmentInterface
	*metav1.DeleteOptions
	ListOptions metav1.ListOptions
}

//VolumeAttachmentGetRequest represents request
type VolumeAttachmentGetRequest struct {
	service_ v1.VolumeAttachmentInterface
	Name     string
	metav1.GetOptions
}

//VolumeAttachmentListRequest represents request
type VolumeAttachmentListRequest struct {
	service_ v1.VolumeAttachmentInterface
	metav1.ListOptions
}

//VolumeAttachmentWatchRequest represents request
type VolumeAttachmentWatchRequest struct {
	service_ v1.VolumeAttachmentInterface
	metav1.ListOptions
}

//VolumeAttachmentPatchRequest represents request
type VolumeAttachmentPatchRequest struct {
	service_     v1.VolumeAttachmentInterface
	Name         string
	Pt           types.PatchType
	Data         []byte
	Subresources []string
}

func init() {
	register(&VolumeAttachmentCreateRequest{})
	register(&VolumeAttachmentUpdateRequest{})
	register(&VolumeAttachmentUpdateStatusRequest{})
	register(&VolumeAttachmentDeleteRequest{})
	register(&VolumeAttachmentDeleteCollectionRequest{})
	register(&VolumeAttachmentGetRequest{})
	register(&VolumeAttachmentListRequest{})
	register(&VolumeAttachmentWatchRequest{})
	register(&VolumeAttachmentPatchRequest{})
}

func (r *VolumeAttachmentCreateRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.VolumeAttachmentInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.VolumeAttachmentInterface", service)
	}
	return nil
}

func (r *VolumeAttachmentCreateRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Create(r.VolumeAttachment)
	return result, err
}

func (r *VolumeAttachmentCreateRequest) GetId() string {
	return "storage/v1.VolumeAttachment.Create"
}

func (r *VolumeAttachmentUpdateRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.VolumeAttachmentInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.VolumeAttachmentInterface", service)
	}
	return nil
}

func (r *VolumeAttachmentUpdateRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Update(r.VolumeAttachment)
	return result, err
}

func (r *VolumeAttachmentUpdateRequest) GetId() string {
	return "storage/v1.VolumeAttachment.Update"
}

func (r *VolumeAttachmentUpdateStatusRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.VolumeAttachmentInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.VolumeAttachmentInterface", service)
	}
	return nil
}

func (r *VolumeAttachmentUpdateStatusRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.UpdateStatus(r.VolumeAttachment)
	return result, err
}

func (r *VolumeAttachmentUpdateStatusRequest) GetId() string {
	return "storage/v1.VolumeAttachment.UpdateStatus"
}

func (r *VolumeAttachmentDeleteRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.VolumeAttachmentInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.VolumeAttachmentInterface", service)
	}
	return nil
}

func (r *VolumeAttachmentDeleteRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	err = r.service_.Delete(r.Name, r.DeleteOptions)
	return result, err
}

func (r *VolumeAttachmentDeleteRequest) GetId() string {
	return "storage/v1.VolumeAttachment.Delete"
}

func (r *VolumeAttachmentDeleteCollectionRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.VolumeAttachmentInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.VolumeAttachmentInterface", service)
	}
	return nil
}

func (r *VolumeAttachmentDeleteCollectionRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	err = r.service_.DeleteCollection(r.DeleteOptions, r.ListOptions)
	return result, err
}

func (r *VolumeAttachmentDeleteCollectionRequest) GetId() string {
	return "storage/v1.VolumeAttachment.DeleteCollection"
}

func (r *VolumeAttachmentGetRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.VolumeAttachmentInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.VolumeAttachmentInterface", service)
	}
	return nil
}

func (r *VolumeAttachmentGetRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Get(r.Name, r.GetOptions)
	return result, err
}

func (r *VolumeAttachmentGetRequest) GetId() string {
	return "storage/v1.VolumeAttachment.Get"
}

func (r *VolumeAttachmentListRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.VolumeAttachmentInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.VolumeAttachmentInterface", service)
	}
	return nil
}

func (r *VolumeAttachmentListRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.List(r.ListOptions)
	return result, err
}

func (r *VolumeAttachmentListRequest) GetId() string {
	return "storage/v1.VolumeAttachment.List"
}

func (r *VolumeAttachmentWatchRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.VolumeAttachmentInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.VolumeAttachmentInterface", service)
	}
	return nil
}

func (r *VolumeAttachmentWatchRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Watch(r.ListOptions)
	return result, err
}

func (r *VolumeAttachmentWatchRequest) GetId() string {
	return "storage/v1.VolumeAttachment.Watch"
}

func (r *VolumeAttachmentPatchRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.VolumeAttachmentInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.VolumeAttachmentInterface", service)
	}
	return nil
}

func (r *VolumeAttachmentPatchRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Patch(r.Name, r.Pt, r.Data, r.Subresources...)
	return result, err
}

func (r *VolumeAttachmentPatchRequest) GetId() string {
	return "storage/v1.VolumeAttachment.Patch"
}
