package endly

import (
	"github.com/viant/toolbox/url"
)

//WorkflowExitRequest represents workflow exit request, to exit a caller workflow
type WorkflowExitRequest struct {
	Source *url.Resource
}

//WorkflowExitResponse represents workflow exit response
type WorkflowExitResponse struct{}
