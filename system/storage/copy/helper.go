package copy

import (
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"strings"
)

// JoinIfNeeded routes parent resource with URI when needed
func JoinIfNeeded(parent *url.Resource, URI string) (result *url.Resource) {
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
		result = url.NewResource(URI)
	} else if !(strings.HasPrefix(URI, "/") || strings.HasPrefix(URI, "$")) {
		var hostname = parent.ParsedURL.Hostname()
		if hostname == "" || hostname == "127.0.0.1" || hostname == "localhost" {
			var candidate = url.NewResource(URI)

			if toolbox.FileExists(candidate.ParsedURL.Path) {
				result = candidate
			}
		}
		if result == nil {
			result = url.NewResource(toolbox.URLPathJoin(parent.URL, URI))
		}

	} else if parent != nil {
		result = url.NewResource(toolbox.URLPathJoin(parent.URL, URI))
	} else {
		result = url.NewResource(URI)
	}
	if strings.HasPrefix(URI, "$") { //has to expand to be re-evaluated
		result.URL = URI
	}

	return result
}
