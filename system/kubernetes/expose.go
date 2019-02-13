package core

import (
	"fmt"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	ServiceV1GeneratorName = "service/v1"
)

type ExposeTemplateParams struct {
	*ExposeRequest
	Ports                 []v1.ServicePort
	Selector              map[string]string
	SessionAffinityConfig *v1.SessionAffinityConfig
}

func IsIntStrEmpty(value intstr.IntOrString) bool {
	return !(value.StrVal != "" || value.IntVal > 0)
}

func (p *ExposeTemplateParams) exposePort(exposingResource *ResourceInfo) error {
	ports := exposingResource.ContainerPorts()
	switch len(ports) {
	case 0:
		if p.Ports[0].Port == 0 {
			return fmt.Errorf("port and targetPort were not set")
		}
	case 1:
		//inherit port, proto, targetPort from source resource if it was not spcified with expose request
		if p.Ports[0].Port == 0 {
			p.Ports[0].Port = ports[0].ContainerPort
		}
		if IsIntStrEmpty(p.Ports[0].TargetPort) {
			p.Ports[0].TargetPort = intstr.FromInt(int(ports[0].ContainerPort))
		}
		if p.Ports[0].Protocol == "" {
			p.Ports[0].Protocol = ports[0].Protocol
		}
	default:
		exposingPort := p.Ports[0].Port
		p.Ports = make([]v1.ServicePort, 1)
		for _, port := range ports {
			servicePort := v1.ServicePort{
				Port:       port.ContainerPort,
				Protocol:   port.Protocol,
				TargetPort: intstr.FromInt(int(ports[0].ContainerPort)),
			}
			if exposingPort != 0 {
				servicePort.Port = exposingPort
			}
			p.Ports = append(p.Ports, servicePort)
		}
	}
	return nil
}

func (p *ExposeTemplateParams) Apply(source *ResourceInfo) error {
	if p.SessionAffinity == "ClientIP" {
		defaultTs := int32(10800)
		p.SessionAffinityConfig = &v1.SessionAffinityConfig{
			ClientIP: &v1.ClientIPConfig{
				TimeoutSeconds: &defaultTs,
			},
		}
	}
	err := p.exposePort(source)
	if err != nil {
		return err
	}
	return p.setMatchingLabels(source)
}

func (p *ExposeTemplateParams) setMatchingLabels(info *ResourceInfo) error {
	p.Selector = info.Labels
	if len(p.Labels) == 0 {
		p.Labels = info.Labels
	}
	return nil
}

func NewExposeTemplateParams(source *ResourceInfo, request *ExposeRequest) (*ExposeTemplateParams, error) {
	result := &ExposeTemplateParams{
		ExposeRequest: request,
		Ports:         make([]v1.ServicePort, 1),
	}
	if request.Port > 0 {
		result.Ports[0].Port = request.Port
	}
	if request.TargetPort != "" {
		result.Ports[0].TargetPort = intstr.Parse(request.TargetPort)
	}
	if request.Protocol != "" {
		result.Ports[0].Protocol = request.Protocol
	}
	return result, result.Apply(source)
}
