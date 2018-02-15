package endly

import (
	"github.com/viant/toolbox/url"
	"github.com/pkg/errors"
)

//BuildLoadMetaRequest represents a loading BuildMeta request
type BuildLoadMetaRequest struct {
	Source *url.Resource `required:"true" description:"URL with build meta JSON"`
}


//Validate checks if request is valid
func (r *BuildLoadMetaRequest) Validate() error {
	if r.Source == nil {
		return errors.New("source was empty")
	}
	return nil
}


//BuildLoadMetaResponse represents build meta response.
type BuildLoadMetaResponse struct {
	Meta *BuildMeta //url to size
}
