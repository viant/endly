package run

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/gcp"
	"google.golang.org/api/run/v1alpha1"
	"strings"
)

const (
	apiVersion = "serving.knative.dev/v1alpha1"
	kind       = "Service"
	nonceLabel = "client.knative.dev/nonce"
	userImage  = "client.knative.dev/user-image"
	memory     = "memory"
)

//DeployRequest represents deploy request
type DeployRequest struct {
	Name            string
	Namespace       string
	Public          bool `description:"allows unauthenticated access"`
	Concurrency     int
	Connectivity    string `description:"valid values external or internal"`
	Image           string
	Memory          string
	Region          string
	Cluster         string
	ClusterLocation string
	Replace         bool
	Environments    map[string]string
	Members         []string `description:"members with roles/run.invoker role"`
	parent          string
	resource        string
}

type DeployResponse struct {
	Endpoint string
	*run.Configuration
}

//GetServiceRequest represents getService service request
type GetServiceRequest struct {
	Name string
	uri  string
}

//GetServiceResponse represents getService response
type GetServiceResponse struct {
	*run.Service
}

//GetServiceRequest represents getService service request
type GetConfigurationRequest struct {
	Name string
	uri  string
}

//GetServiceResponse represents getService response
type GetConfigurationResponse struct {
	*run.Configuration
}

//Init initializes request
func (r *DeployRequest) Init() error {
	if r.Namespace == "" {
		r.Namespace = "$gcp.projectID"
	}
	if r.Region == "" {
		r.Region = gcp.DefaultRegion
	}
	if r.Name == "" && r.Image != "" {
		if imageNamePosition := strings.LastIndex(r.Image, "/"); imageNamePosition != -1 {
			name := string(r.Image[imageNamePosition+1:])
			if versionPosition := strings.Index(name, ":"); versionPosition != -1 {
				name = string(name[:versionPosition])
			}
			r.Name = name
		}
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
			Annotations:  make(map[string]string),
			Labels:       make(map[string]string),
			Initializers: &run.Initializers{},
			Name:         r.Name,
			Namespace:    gcp.ExpandMeta(context, r.Namespace), //project
		},
		Spec: &run.ServiceSpec{
			RunLatest: &run.ServiceSpecRunLatest{
				Configuration: &run.ConfigurationSpec{
					RevisionTemplate: &run.RevisionTemplate{
						Metadata: &run.ObjectMeta{
							Labels: map[string]string{
								nonceLabel: strings.ToLower(generateRandomASCII(10)),
							},
						},
						Spec: &run.RevisionSpec{
							Container: &run.Container{
								Image: r.Image,
							},
						},
					},
				},
			},
		},
		Status: &run.ServiceStatus{},
	}

	if r.Memory != "" {
		result.Spec.RunLatest.Configuration.RevisionTemplate.Spec.Container.Resources = &run.ResourceRequirements{
			Limits:   make(map[string]string),
			Requests: make(map[string]string),
		}
		result.Spec.RunLatest.Configuration.RevisionTemplate.Spec.Container.Resources.Limits[memory] = r.Memory
	}
	if r.Concurrency > 0 {
		result.Spec.RunLatest.Configuration.RevisionTemplate.Spec.ContainerConcurrency = int64(r.Concurrency)
	}
	result.Metadata.Annotations[userImage] = r.Image
	return result, nil
}
