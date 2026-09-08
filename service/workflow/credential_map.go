package workflow

import (
	"fmt"

	"github.com/viant/endly"
	"github.com/viant/endly/model"
	"github.com/viant/endly/service/credential"
)

func extractCredentialMap(request *RunRequest, workflow *model.Workflow) map[string]string {
	if request != nil && request.Inlined != nil && len(request.Inlined.CredentialMap) > 0 {
		return request.Inlined.CredentialMap
	}
	if workflow != nil && len(workflow.CredentialMap) > 0 {
		return workflow.CredentialMap
	}
	return nil
}

func credentialMapSource(request *RunRequest, workflow *model.Workflow) string {
	if request != nil && request.AssetURL != "" {
		return request.AssetURL
	}
	if workflow != nil && workflow.Source != nil && workflow.Source.URL != "" {
		return workflow.Source.URL
	}
	if request != nil && request.Name != "" {
		return request.Name
	}
	return "unknown workflow"
}

// applyCredentialMap registers credentialMap from the root workflow, or errors if a nested workflow defines one.
func applyCredentialMap(context *endly.Context, request *RunRequest, workflow *model.Workflow, isRoot bool) error {
	m := extractCredentialMap(request, workflow)
	if len(m) == 0 {
		return nil
	}
	source := credentialMapSource(request, workflow)
	if !isRoot {
		return fmt.Errorf("credentialMap is only allowed on the root workflow; nested workflow %s must not define credentialMap", source)
	}
	svc, ok := context.Secrets.(*credential.Service)
	if !ok {
		return fmt.Errorf("credentialMap requires endly credential.Service (workflow %s)", source)
	}
	svc.SetCredentialMap(m)
	return nil
}
