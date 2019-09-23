package storage

import (
	"errors"
	"github.com/viant/endly/system/storage/transfer"
	"github.com/viant/endly/testing/validator"
	"github.com/viant/toolbox/url"
	"github.com/viant/afs/asset"
)


//DownloadRequest represents a resources download request, it downloads source into context.state target key
type ListRequest struct {
	Source  *url.Resource `required:"true" description:"source asset or directory"`
	Matcher *transfer.Matcher
	IncludeContent bool
	Expect interface{}
}


//DownloadResponse represents a download response
type ListResponse struct {
	Assets []*asset.Resource
	Assert      *validator.AssertResponse
}

//Validate checks if request is valid
func (r *ListRequest) Validate() error {
	if r.Source == nil {
		return errors.New("source was empty")
	}
	return nil
}
