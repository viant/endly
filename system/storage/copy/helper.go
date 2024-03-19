package copy

import (
	"github.com/viant/afs/file"
	"github.com/viant/afs/url"
	"github.com/viant/endly/model/location"
)

// JoinIfNeeded routes parent resource with URI when needed
func JoinIfNeeded(parent *location.Resource, URI string) (result *location.Resource) {
	schema := url.Scheme(URI, file.Scheme)
	switch schema {
	case file.Scheme:
		if url.Path(URI) == parent.Path() {
			return parent
		}
		return parent.Join(URI)
	default:
		result = location.NewResource(URI)
	}
	return result
}
