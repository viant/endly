package run

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/viant/endly"
	"github.com/viant/endly/service/system/cloud/gcp"
	"github.com/viant/toolbox"
	"google.golang.org/api/run/v1"
	"strings"
)

const (
	apiVersion     = "serving.knative.dev/v1"
	kind           = "Service"
	clientName     = "run.googleapis.com/client-name"
	clientImage    = "client.knative.dev/user-image"
	autoScalingMax = "autoscaling.knative.dev/maxScale"
	client         = "run.googleapis.com/client-name"

	memory = "memory"
	cpu    = "cpu"
)

// DeployRequest represents deploy request
type DeployRequest struct {
	Name           string
	Namespace      string
	Public         bool `description:"allows unauthenticated access"`
	Concurrency    int
	TimeoutSeconds int
	Connectivity   string `description:"valid values external or internal"`
	Container      *run.Container
	Env            map[string]string
	Image          string
	MemoryMb       int
	CPU            int
	Port           int
	Region         string
	MaxAutoScale   int
	ServiceAccount string
	Replace        bool
	Members        []string `description:"members with roles/run.invoker role"`
	parent         string
	resource       string
}

type DeployResponse struct {
	Endpoint string
	*run.Configuration
}

// GetServiceRequest represents getService service request
type GetServiceRequest struct {
	Name string
	uri  string
}

// GetServiceResponse represents getService response
type GetServiceResponse struct {
	*run.Service
}

// GetServiceRequest represents getService service request
type GetConfigurationRequest struct {
	Name string
	uri  string
}

// GetServiceResponse represents getService response
type GetConfigurationResponse struct {
	*run.Configuration
}

// Init initializes request
func (r *DeployRequest) Validate() error {
	if r.Container == nil && r.Image == "" {
		return errors.Errorf("container was empty")
	}
	if r.Container.Image == "" && r.Image == "" {
		return errors.Errorf("container.Image was empty")
	}
	return nil
}

// Init initializes request
func (r *DeployRequest) Init() error {
	if r.Namespace == "" {
		r.Namespace = "$gcp.projectID"
	}
	if r.MemoryMb == 0 {
		r.MemoryMb = 128
	}
	if r.CPU == 0 {
		r.CPU = 1000
	}
	if r.Port == 0 {
		r.Port = 8080
	}
	if r.MaxAutoScale == 0 {
		r.MaxAutoScale = 1000
	}
	if r.Concurrency == 0 {
		r.Concurrency = 1000
	}
	if r.Region == "" {
		r.Region = gcp.DefaultRegion
	}

	if r.Container == nil && r.Image != "" {
		r.Container = &run.Container{
			Image: r.Image,
		}
	}
	if len(r.Env) > 0 {
		if len(r.Container.Env) == 0 {
			r.Container.Env = make([]*run.EnvVar, 0)
			for k, v := range r.Env {
				r.Container.Env = append(r.Container.Env, &run.EnvVar{
					Name:  k,
					Value: v,
				})
			}
		}
	}
	if r.Port > 0 {
		if len(r.Container.Ports) == 0 {
			r.Container.Ports = make([]*run.ContainerPort, 0)
			r.Container.Ports = append(r.Container.Ports, &run.ContainerPort{
				ContainerPort: int64(r.Port),
			})
		}

	}

	if r.Name == "" && r.Container.Image != "" {
		if imageNamePosition := strings.LastIndex(r.Container.Image, "/"); imageNamePosition != -1 {
			name := r.Container.Image[imageNamePosition+1:]
			if versionPosition := strings.Index(name, ":"); versionPosition != -1 {
				name = name[:versionPosition]
			}
			r.Name = name
		}
	}
	if r.Container.Resources == nil {
		r.Container.Resources = &run.ResourceRequirements{}
	}
	if len(r.Container.Resources.Limits) == 0 {
		r.Container.Resources.Limits = map[string]string{}
	}
	if len(r.Container.Resources.Requests) == 0 {
		r.Container.Resources.Requests = map[string]string{}
	}
	if r.MemoryMb > 0 {
		r.Container.Resources.Limits[memory] = fmt.Sprintf("%vMi", r.MemoryMb)
	}
	if r.CPU > 0 {
		r.Container.Resources.Limits[cpu] = fmt.Sprintf("%vm", r.CPU)
	}
	if r.TimeoutSeconds == 0 {
		r.TimeoutSeconds = 300
	}
	r.parent = "namespaces/${gcp.projectID}"
	r.resource = fmt.Sprintf("projects/${gcp.projectID}/locations/%v/services/%v", r.Region, r.Name)
	if r.Public && len(r.Members) == 0 {
		r.Members = []string{"allUsers"}
	}
	return nil
}

func (r *GetServiceRequest) Init() error {
	r.uri = fmt.Sprintf("namespaces/${gcp.projectID}/services/%v", r.Name)
	return nil
}

func (r *GetConfigurationRequest) Init() error {
	r.uri = fmt.Sprintf("namespaces/${gcp.projectID}/configurations/%v", r.Name)
	return nil
}

func (r *DeployRequest) Service(context *endly.Context) (*run.Service, error) {

	result := &run.Service{
		ApiVersion: apiVersion,
		Kind:       kind,
		Metadata: &run.ObjectMeta{
			Annotations: map[string]string{
				clientImage: r.Container.Image,
				clientName:  endly.AppName,
			},
			Labels:    make(map[string]string),
			Name:      r.Name,
			Namespace: gcp.ExpandMeta(context, r.Namespace), //project
		},
		Spec: &run.ServiceSpec{
			Template: &run.RevisionTemplate{
				Metadata: &run.ObjectMeta{
					Annotations: map[string]string{
						autoScalingMax: toolbox.AsString(r.MaxAutoScale),
						clientImage:    r.Container.Image,
						clientName:     endly.AppName,
					},
					Labels: make(map[string]string),
					Name:   r.Name + "-" + strings.ToLower(generateRandomASCII(10)),
				},
				Spec: &run.RevisionSpec{
					ContainerConcurrency: int64(r.Concurrency),
					ServiceAccountName:   r.ServiceAccount,
					TimeoutSeconds:       int64(r.TimeoutSeconds),
					Containers:           []*run.Container{r.Container},
				},
			},
		},
		Status: &run.ServiceStatus{},
	}

	return result, nil
}
