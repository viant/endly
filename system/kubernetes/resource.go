package core

import (
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
	"gopkg.in/yaml.v2"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path"
	"strings"
)

func ProcessResource(context *endly.Context, expand bool, resource *url.Resource, reverse bool, handler func(meta *ResourceMeta, data map[string]interface{}) error) error {
	resource, err := context.ExpandResource(resource)
	if err != nil {
		return err
	}
	state := context.State()
	meta := &ResourceMeta{}
	var data = make(map[string]interface{})
	ext := path.Ext(resource.URL)
	if ext == ".json" {
		if err = resource.Decode(&data); err != nil {
			return err
		}
		if expand {
			data = toolbox.AsMap(state.Expand(data))
		}
		if err = converter.AssignConverted(meta, data); err == nil {
			err = handler(meta, data)
		}
		return err
	}

	text, err := resource.DownloadText()
	if err != nil {
		return err
	}
	if expand {
		text= state.ExpandAsText(text)
	}

	docs := strings.Split(text, "---\n")
	if len(docs) == 0 {
		return nil
	}
	if reverse {
		for i := len(docs) - 1; i >= 0; i-- {
			if err = handleResource(docs[i], expand, state, data, handler); err != nil {
				return err
			}

		}
		return nil
	}

	for _, doc := range docs {
		if err = handleResource(doc, expand, state, data, handler); err != nil {
			return err
		}
	}
	return nil
}

func handleResource(doc string, expand bool, state data.Map, data map[string]interface{}, handler func(meta *ResourceMeta, data map[string]interface{}) error) error {
	var meta = &ResourceMeta{}
	if err := yaml.Unmarshal([]byte(doc), &data); err != nil {
		return err
	}
	normalized, err := toolbox.NormalizeKVPairs(data)
	if err == nil {
		data = toolbox.AsMap(normalized)
	}
	if expand {
		data = toolbox.AsMap(state.Expand(data))
	}
	if err = converter.AssignConverted(meta, data); err == nil && len(data) > 0 {
		err = handler(meta, data)
	}
	return err
}

//ResourceMeta represents a k8 resource meta
type ResourceMeta struct {
	metav1.TypeMeta
	Metadata metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
}

// ResourceCondition contains details for the current condition
type ResourceCondition struct {
	Type    string
	Status  string
	Reason  string
	Message string
}

type ResourceStatus struct {
	Conditions               []ResourceCondition
	PublishNotReadyAddresses *bool
}

//ResourceInfo represents generic  resource info
type ResourceInfo struct {
	metav1.TypeMeta   `yaml:",inline"`
	metav1.ObjectMeta `yaml:"metadata"`
	Spec              interface{}
	Status            interface{}
	Raw               interface{} `yaml:"-"`
}

func (i *ResourceInfo) ResourceStatus() *ResourceStatus {
	var status = &ResourceStatus{}
	if i.Status != nil {
		_ = converter.AssignConverted(status, i.Status)
	}
	return status
}
func (i *ResourceInfo) ContainerPorts() []v1.ContainerPort {
	containers := i.Containers()
	if len(containers) == 0 {
		return nil
	}
	return containers[0].Ports
}

func (i *ResourceInfo) Containers() []v1.Container {
	if i.Spec == nil {
		return nil
	}
	specMap := data.Map(toolbox.AsMap(i.Spec))
	containersValue, ok := specMap.GetValue("template.spec.containers")
	if !ok {
		return nil
	}
	var containers = make([]v1.Container, 0)
	_ = converter.AssignConverted(&containers, containersValue)
	return containers
}

func (i *ResourceInfo) IsReady() bool {
	if i.Status == nil {
		return false
	}
	status := i.ResourceStatus()
	if status.PublishNotReadyAddresses != nil {
		return !*status.PublishNotReadyAddresses
	}
	if len(status.Conditions) == 0 {
		return false
	}
	for _, condition := range status.Conditions {
		if condition.Status == "False" {
			return false
		}
	}
	return true
}

//ResourcesMetaInfo represents resource meta info
type ResourcesMetaInfo struct {
	*ResourceMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Items         []*ResourceMeta `json:"items,omitempty" yaml:"items,omitempty"`
}

//ToResourceMetas converts *ResourceInfo slice to []*ResourceMeta
func ToResourceMetas(items []*ResourceInfo) *ResourcesMetaInfo {
	var result = &ResourcesMetaInfo{
		Items: make([]*ResourceMeta, 0),
	}
	switch len(items) {
	case 0:
	case 1:
		result.ResourceMeta = &ResourceMeta{TypeMeta: items[0].TypeMeta, Metadata: items[0].ObjectMeta}
	default:
		for _, item := range items {
			result.Items = append(result.Items, &ResourceMeta{TypeMeta: item.TypeMeta, Metadata: item.ObjectMeta})
		}
		result.ResourceMeta = nil
	}
	return result
}
