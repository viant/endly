package endly

import "github.com/viant/toolbox/url"

//WorkflowRegisterRequest represents workflow register request
type WorkflowRegisterRequest struct {
	Workflow *Workflow
}

//WorkflowRegisterResponse represents workflow register response
type WorkflowRegisterResponse struct {
	Source *url.Resource
}
