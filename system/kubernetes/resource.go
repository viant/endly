package core

import (
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path"
	"strings"
)

func ProcessResource(context *endly.Context, resource *url.Resource, handler func(meta *metav1.TypeMeta, data map[string]interface{}) error) error {
	resource, err := context.ExpandResource(resource)
	if err != nil {
		return err
	}
	var meta *metav1.TypeMeta
	var data = make(map[string]interface{})

	ext := path.Ext(resource.URL)
	if ext == ".json" {
		if err = resource.Decode(&data); err != nil {
			return err
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
	docs := strings.Split(text, "---\n")
	for _, doc := range docs {
		if err = yaml.Unmarshal([]byte(doc), &data); err != nil {
			return err
		}
		if normalized, err := toolbox.NormalizeKVPairs(data); err == nil {
			data = toolbox.AsMap(normalized)
		}
		if err = converter.AssignConverted(meta, data); err == nil {
			err = handler(meta, data)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
