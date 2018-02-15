package endly

import "github.com/viant/toolbox/url"

//BuildLoadMetaRequest represents a loading BuildMeta request
type BuildLoadMetaRequest struct {
	Source *url.Resource `required:"true" description:"URL with build meta JSON"`
}

//BuildLoadMetaResponse represents build meta response.
type BuildLoadMetaResponse struct {
	Meta *BuildMeta //url to size
}
