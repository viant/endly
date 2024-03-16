package copy

import (
	"github.com/viant/endly/model/location"
	"github.com/viant/toolbox"
	"strings"
)

// JoinIfNeeded routes parent resource with URI when needed
func JoinIfNeeded(parent *location.Resource, URI string) (result *location.Resource) {
	defer func() {
		if parent != nil {
			if result.Credentials == "" {
				result.Credentials = parent.Credentials
			}
			if result.CustomKey == nil {
				result.CustomKey = parent.CustomKey
			}
		}
	}()
	if strings.Contains(URI, ":/") {
		result = location.NewResource(URI)
	} else if !(strings.HasPrefix(URI, "/") || strings.HasPrefix(URI, "$")) {
		var hostname = parent.Hostname()
		if hostname == "" || hostname == "127.0.0.1" || hostname == "localhost" {
			var candidate = location.NewResource(URI)

			if toolbox.FileExists(candidate.Path()) {
				result = candidate
			}
		}
		if result == nil {
			result = location.NewResource(toolbox.URLPathJoin(parent.URL, URI))
		}

	} else if parent != nil {
		result = location.NewResource(toolbox.URLPathJoin(parent.URL, URI))
	} else {
		result = location.NewResource(URI)
	}
	if strings.HasPrefix(URI, "$") { //has to expand to be re-evaluated
		result.URL = URI
	}

	return result
}
