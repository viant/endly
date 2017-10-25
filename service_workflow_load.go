package endly

import (
	"github.com/viant/toolbox/url"
)

//WorkflowLoadRequest represents workflow load request from the specified source
type WorkflowLoadRequest struct {
	Source *url.Resource
}

//WorkflowLoadResponse represents loaded workflow
type WorkflowLoadResponse struct {
	Workflow *Workflow
}
