package core

import (
	"github.com/viant/endly/system/kubernetes/shared"
	"github.com/viant/toolbox/url"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

type GetRequest struct {
	Name string
	metav1.ListOptions
	OutputTemplate string
	OutputPaths    map[string]string
	apiKindMethod_ string
	multiItem      bool
}

type GetResponse interface{}

type CreateRequest struct {
	*url.Resource
}

type CreateResponse struct {
	Items []interface{}
}

type ApplyRequest struct {
	Source url.Resource
	metav1.GetOptions
}

func (r *GetRequest) Init() (err error) {
	if r.Kind == "" && r.Name != "" {
		if parts := strings.Split(r.Name, "/"); len(parts) == 2 {
			r.Kind = parts[0]
			r.Name = parts[1]
		}
	}
	r.Kind = strings.Title(r.Kind)
	if r.Name == "" {
		r.apiKindMethod_ = "List"
		r.multiItem = true
	} else {
		r.apiKindMethod_ = "Get"
	}

	if r.APIVersion == "" && r.Kind != "" {
		if r.APIVersion, err = shared.LookupAPIVersion(r.Kind); err != nil {
			return err
		}
	}
	return nil
}
