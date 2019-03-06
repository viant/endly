package core

import (
	"fmt"
	"github.com/viant/endly/system/kubernetes/shared"
	"github.com/viant/toolbox/url"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/core"

	"strings"
)

type ResourceInfoResponse struct {
	Items []*ResourceInfo
}

//GetRequest represents get request
type GetRequest struct {
	Name string
	metav1.ListOptions
	Describe       bool `description:"describe flag control output"`
	apiKindMethod_ string
	multiItem      bool
	kinds          []string
}

//GetResponse represents get response
type GetResponse ResourceInfoResponse

//CreateRequest represents create request
type CreateRequest struct {
	*ResourceMeta
	*url.Resource
	Expand bool `description:"flag to expand resource with $ expression"`
}

//CreateResponse represents create response
type CreateResponse ResourceInfoResponse

//DeleteRequest represents delete response
type DeleteRequest struct {
	Name            string
	LabelSelector   string
	metav1.TypeMeta `json:",inline"`
	*url.Resource
}

//DeleteResponse represents delete response
type DeleteResponse ResourceInfoResponse

//ApplyRequest represents apply request
type ApplyRequest struct {
	*url.Resource
	*ResourceMeta
	Expand bool `description:"flag to expand resource with $ expression"`
}

//ApplyResponse represents apply response
type ApplyResponse ResourceInfoResponse

//RunRequest represents run request
type RunRequest struct {
	Expose          bool
	Replicas        int
	Template        string
	Schedule        string
	Labels          map[string]string
	Name            string             `description:"metadata.name"`
	RestartPolicy   core.RestartPolicy `description:"spec.restartPolicy"`
	DNSPolicy       v1.DNSPolicy       `description:"spec.dNSPolicy"`
	ServiceAccount  string             `description:"spec.serviceAccountName"`
	ImagePullPolicy core.PullPolicy    `description:"spec.containers[].imagePullPolicy"`
	Image           string             `description:"spec.containers[].image"`
	Commands        []string           `description:"spec.containers[].commands"`
	Args            []string           `description:"spec.containers[].args"`
	Env             map[string]string  `description:"spec.containers[].env"`
	Limits          map[string]string  `description:"spec.containers[].resources.limits"`
	Requests        map[string]string  `description:"spec.containers[].resources.requests"`
	Port            int                `description:"spec.containers[].ports[].containerPort"`
	HostPort        int                `description:"spec.containers[].ports[].hostPort"`
}

//RunResponse represents run response
type RunResponse ResourceInfoResponse

//ExposeRequest represent expose request
type ExposeRequest struct {
	Resource     string `description:"a target resource name to be exposed"`
	ResourceKind string `description:"optional target resource kind"`

	Name           string      `description:"metadata.name"`
	Protocol       v1.Protocol `description:"spec.ports[].protocol"`
	Port           int32       `description:"spec.ports[].port: expose port"`
	TargetPort     string      `description:"spec.ports[].targetPort: name or number for the port on the container that the service should direct traffic to"`
	Type           string      `description:"spec.type: for this service: ClusterIP, NodePort, LoadBalancer, or ExternalName"`
	LoadBalancerIP string      `description:"spec.loadBalancerIP: IP to assign to the LoadBalancer"`

	Labels              map[string]string `description:"spec.labels"`
	ExternalIPs         string            `description:"spec.ExternalIPs"`
	SessionAffinity     string            `description:"spec.SessionAffinity: if non empty: 'None', 'ClientIP'"`
	ClusterIP           string            `description:"spec.ClusterIP: to be assigned to the service. Leave empty to auto-allocate, or set to 'None' to create a headless service"`
	ExternalName        string            `description:"spec.externalName"`
	HealthCheckNodePort int               `description:"spec.HealthCheckNodePort"`

	kinds []string
}

//ExposeResponse represent expose response
type ExposeResponse ResourceInfoResponse

type CopyRequest struct {
}

type CopyResponse struct {
}

type ExecRequest struct {
}

type ExecResponse struct {
}

//Init initialises request
func (r *ExposeRequest) Init() error {
	if r.Protocol == "" {
		r.Protocol = "TCP"
	}
	if r.Type == "" {
		r.Type = "ClusterIP"
	}
	if len(r.Labels) == 0 {
		r.Labels = make(map[string]string)
	}
	if strings.Contains(r.Resource, "/") {
		pair := strings.SplitN(r.Resource, "/", 2)
		r.ResourceKind = pair[0]
		r.Resource = pair[1]
	}

	if r.ResourceKind == "" {
		r.kinds = shared.MatchedMetaTypes("Deployment", "Service", "ReplicaSet", "ReplicationController", "Pod")
	} else {
		r.kinds = make([]string, 0)
		for _, kind := range strings.Split(r.ResourceKind, ",") {
			r.kinds = append(r.kinds, strings.TrimSpace(kind))
		}
	}

	if r.Name == "" { //inherit nme from source resource
		r.Name = r.Resource
	}

	return nil
}

//Validate checks if request is valid
func (r *ExposeRequest) Validate() error {
	if r.Resource == "" {
		return fmt.Errorf("resource was empty")
	}

	return nil
}

func (r *GetRequest) Init() (err error) {
	if r.Kind == "" && r.Name != "" {
		if parts := strings.Split(r.Name, "/"); len(parts) == 2 {
			r.Kind = parts[0]
			r.Name = parts[1]
		}
	}
	if r.Name == "" {
		r.apiKindMethod_ = "List"
		r.multiItem = true
	} else {
		r.apiKindMethod_ = "Get"
	}
	if r.Kind != "" && len(r.kinds) == 0 {
		r.kinds = make([]string, 0)
		if strings.Contains(r.Kind, ",") {
			for _, item := range strings.Split(r.Kind, "") {
				r.kinds = append(r.kinds, strings.TrimSpace(item))
			}
			r.kinds = shared.MetaTypes()
		} else if strings.Contains(r.Kind, "*") {
			for _, item := range strings.Split(r.Kind, "") {
				r.kinds = append(r.kinds, strings.TrimSpace(item))
			}
			r.kinds = shared.MetaTypes()
		} else {
			r.kinds = append(r.kinds, r.Kind)
		}
	}
	return nil
}

func (r *GetRequest) Validate() (err error) {
	if r.Kind == "" {
		return fmt.Errorf("kind was empty, consider using placeholder kind='*' for global search")
	}
	return nil
}

func (r *RunRequest) Init() error {
	if r.ImagePullPolicy == "" {
		r.ImagePullPolicy = core.PullAlways
	}

	if len(r.Labels) == 0 {
		r.Labels = make(map[string]string)
	}
	if r.Replicas == 0 {
		r.Replicas = 1
	}
	if r.DNSPolicy == "" {
		r.DNSPolicy = v1.DNSClusterFirst
	}
	if r.RestartPolicy == "" {
		r.RestartPolicy = core.RestartPolicyAlways
	}

	r.Labels["run"] = r.Name
	if r.Template == "" && r.Schedule != "" {
		r.Template = CronJobV1Beta1GeneratorName
	}
	if r.Template == "" {
		switch r.RestartPolicy {
		case core.RestartPolicyAlways:
			r.Template = DeploymentAppsV1GeneratorName
		case core.RestartPolicyOnFailure:
			r.Template = JobV1GeneratorName
		case core.RestartPolicyNever:
			r.Template = RunPodV1GeneratorName
		}
	}
	return nil
}

func (r *RunRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name was empty")
	}
	if r.HostPort > 0 && r.Port == 0 {
		return fmt.Errorf("hostPort required port")
	}
	return nil
}

func (r *CreateRequest) Init() (err error) {
	if r.Resource != nil {
		if err = r.Resource.Init(); err != nil {
			return err
		}
	}
	return nil
}

func (r *DeleteRequest) Init() (err error) {
	if r.Resource != nil {
		if err = r.Resource.Init(); err != nil {
			return err
		}
	}
	return nil
}

func (r *DeleteRequest) AsGetRequest() *GetRequest {
	result := &GetRequest{
		Name: r.Name,
	}
	if r.Kind == "*" {
		result.kinds = shared.MatchedMetaTypes("Deployment", "Service", "ReplicaSet", "ReplicationController", "Pod", "Job", "CronJob", "Endpoints")
	}
	result.TypeMeta = r.TypeMeta
	result.LabelSelector = r.LabelSelector
	return result
}

//Init initializes request
func (r *ApplyRequest) Init() error {
	return nil
}
