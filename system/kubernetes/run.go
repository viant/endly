package core

import (
	"github.com/viant/toolbox"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sort"
)

const (
	RunV1GeneratorName            = "run/v1"
	RunPodV1GeneratorName         = "run-pod/v1"
	DeploymentAppsV1GeneratorName = "deployment/apps.v1"
	JobV1GeneratorName            = "job/v1"
	CronJobV1Beta1GeneratorName   = "cronjob/v1beta1"
)

//RunTemplateParams represents run template parameters
type RunTemplateParams struct {
	*RunRequest
	Ports         []*v1.ContainerPort
	Resources     []*v1.ResourceRequirements
	LabelSelector *metav1.LabelSelector // {MatchLabels: labels}
	Envs          []v1.EnvVar
}

//NewRunTemplateParams create a new run template parameters for supplied run request
func NewRunTemplateParams(request *RunRequest) (*RunTemplateParams, error) {
	result := &RunTemplateParams{
		RunRequest:    request,
		Ports:         make([]*v1.ContainerPort, 0),
		Resources:     make([]*v1.ResourceRequirements, 0),
		LabelSelector: &metav1.LabelSelector{},
		Envs:          make([]v1.EnvVar, 0),
	}

	if len(result.Commands) == 0 {
		result.Commands = []string{}
	}
	if result.Port > 0 {
		result.Ports = append(result.Ports,
			&v1.ContainerPort{
				ContainerPort: int32(result.Port),
				HostPort:      int32(result.HostPort),
			})
	}
	if len(request.Labels) > 0 {
		result.LabelSelector.MatchLabels = result.Labels
	}
	requirements, err := buildResourceRequirements(request.Requests, request.Limits)
	if err != nil {
		return result, err
	}
	result.Resources = append(result.Resources, requirements)
	if len(request.Env) > 0 {

		keys := toolbox.MapKeysToStringSlice(request.Env)
		sort.Strings(keys)
		for _, k := range keys {
			v := request.Env[k]
			result.Envs = append(result.Envs, v1.EnvVar{
				Name:  k,
				Value: v,
			})
		}
	}
	return result, nil
}
