package endly

import (
	"github.com/viant/toolbox/url"
)

//VcCommitRequest represents a commit request
type VcCommitRequest struct {
	Target  *url.Resource //local code source repo
	Message string        //commit message
}
