package endly

import "github.com/viant/toolbox/url"

//BuildLoadMetaRequest represents a loading BuildMeta request
type BuildLoadMetaRequest struct {
	Resource *url.Resource
}

//BuildLoadMetaResponse represents build meta response.
type BuildLoadMetaResponse struct {
	Loaded map[string]*BuildMeta //url to size
}
