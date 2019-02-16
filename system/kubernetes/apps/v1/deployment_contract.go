package v1



import (
	"fmt"
"errors"
	"k8s.io/client-go/kubernetes/typed/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	vvc "k8s.io/api/apps/v1"	
)

/*autogenerated contract adapter*/

//DeploymentCreateRequest represents request
type DeploymentCreateRequest struct {
  service_ v1.DeploymentInterface
   *vvc.Deployment
}

//DeploymentUpdateRequest represents request
type DeploymentUpdateRequest struct {
  service_ v1.DeploymentInterface
   *vvc.Deployment
}

//DeploymentUpdateStatusRequest represents request
type DeploymentUpdateStatusRequest struct {
  service_ v1.DeploymentInterface
   *vvc.Deployment
}

//DeploymentDeleteRequest represents request
type DeploymentDeleteRequest struct {
  service_ v1.DeploymentInterface
  Name string
   *metav1.DeleteOptions
}

//DeploymentDeleteCollectionRequest represents request
type DeploymentDeleteCollectionRequest struct {
  service_ v1.DeploymentInterface
   *metav1.DeleteOptions
  ListOptions metav1.ListOptions
}

//DeploymentGetRequest represents request
type DeploymentGetRequest struct {
  service_ v1.DeploymentInterface
  Name string
   metav1.GetOptions
}

//DeploymentListRequest represents request
type DeploymentListRequest struct {
  service_ v1.DeploymentInterface
   metav1.ListOptions
}

//DeploymentWatchRequest represents request
type DeploymentWatchRequest struct {
  service_ v1.DeploymentInterface
   metav1.ListOptions
}

//DeploymentPatchRequest represents request
type DeploymentPatchRequest struct {
  service_ v1.DeploymentInterface
  Name string
  Pt types.PatchType
  Data []byte
  Subresources []string
}

//DeploymentGetScaleRequest represents request
type DeploymentGetScaleRequest struct {
  service_ v1.DeploymentInterface
  DeploymentName string
   metav1.GetOptions
}

//DeploymentUpdateScaleRequest represents request
type DeploymentUpdateScaleRequest struct {
  service_ v1.DeploymentInterface
  DeploymentName string
   *autoscalingv1.Scale
}


func init() {
	register(&DeploymentCreateRequest{})
	register(&DeploymentUpdateRequest{})
	register(&DeploymentUpdateStatusRequest{})
	register(&DeploymentDeleteRequest{})
	register(&DeploymentDeleteCollectionRequest{})
	register(&DeploymentGetRequest{})
	register(&DeploymentListRequest{})
	register(&DeploymentWatchRequest{})
	register(&DeploymentPatchRequest{})
	register(&DeploymentGetScaleRequest{})
	register(&DeploymentUpdateScaleRequest{})
}


func (r * DeploymentCreateRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.DeploymentInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.DeploymentInterface", service)
	}
	return nil
}

func (r * DeploymentCreateRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Create(r.Deployment)
	return result, err	
}

func (r * DeploymentCreateRequest) GetId() string {
	return "apps/v1.Deployment.Create";	
}

func (r * DeploymentUpdateRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.DeploymentInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.DeploymentInterface", service)
	}
	return nil
}

func (r * DeploymentUpdateRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Update(r.Deployment)
	return result, err	
}

func (r * DeploymentUpdateRequest) GetId() string {
	return "apps/v1.Deployment.Update";	
}

func (r * DeploymentUpdateStatusRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.DeploymentInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.DeploymentInterface", service)
	}
	return nil
}

func (r * DeploymentUpdateStatusRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.UpdateStatus(r.Deployment)
	return result, err	
}

func (r * DeploymentUpdateStatusRequest) GetId() string {
	return "apps/v1.Deployment.UpdateStatus";	
}

func (r * DeploymentDeleteRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.DeploymentInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.DeploymentInterface", service)
	}
	return nil
}

func (r * DeploymentDeleteRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	err = r.service_.Delete(r.Name,r.DeleteOptions)
	return result, err	
}

func (r * DeploymentDeleteRequest) GetId() string {
	return "apps/v1.Deployment.Delete";	
}

func (r * DeploymentDeleteCollectionRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.DeploymentInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.DeploymentInterface", service)
	}
	return nil
}

func (r * DeploymentDeleteCollectionRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	err = r.service_.DeleteCollection(r.DeleteOptions,r.ListOptions)
	return result, err	
}

func (r * DeploymentDeleteCollectionRequest) GetId() string {
	return "apps/v1.Deployment.DeleteCollection";	
}

func (r * DeploymentGetRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.DeploymentInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.DeploymentInterface", service)
	}
	return nil
}

func (r * DeploymentGetRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Get(r.Name,r.GetOptions)
	return result, err	
}

func (r * DeploymentGetRequest) GetId() string {
	return "apps/v1.Deployment.Get";	
}

func (r * DeploymentListRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.DeploymentInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.DeploymentInterface", service)
	}
	return nil
}

func (r * DeploymentListRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.List(r.ListOptions)
	return result, err	
}

func (r * DeploymentListRequest) GetId() string {
	return "apps/v1.Deployment.List";	
}

func (r * DeploymentWatchRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.DeploymentInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.DeploymentInterface", service)
	}
	return nil
}

func (r * DeploymentWatchRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Watch(r.ListOptions)
	return result, err	
}

func (r * DeploymentWatchRequest) GetId() string {
	return "apps/v1.Deployment.Watch";	
}

func (r * DeploymentPatchRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.DeploymentInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.DeploymentInterface", service)
	}
	return nil
}

func (r * DeploymentPatchRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.Patch(r.Name,r.Pt,r.Data,r.Subresources...)
	return result, err	
}

func (r * DeploymentPatchRequest) GetId() string {
	return "apps/v1.Deployment.Patch";	
}

func (r * DeploymentGetScaleRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.DeploymentInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.DeploymentInterface", service)
	}
	return nil
}

func (r * DeploymentGetScaleRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.GetScale(r.DeploymentName,r.GetOptions)
	return result, err	
}

func (r * DeploymentGetScaleRequest) GetId() string {
	return "apps/v1.Deployment.GetScale";	
}

func (r * DeploymentUpdateScaleRequest) SetService(service interface{}) error {
	var ok bool
	if r.service_, ok = service.(v1.DeploymentInterface); !ok {
		return fmt.Errorf("invalid service type: %T, expected: v1.DeploymentInterface", service)
	}
	return nil
}

func (r * DeploymentUpdateScaleRequest) Call() (result interface{}, err error) {
	if r.service_ == nil {
		return nil, errors.New("service was empty")
	}
	result, err = r.service_.UpdateScale(r.DeploymentName,r.Scale)
	return result, err	
}

func (r * DeploymentUpdateScaleRequest) GetId() string {
	return "apps/v1.Deployment.UpdateScale";	
}