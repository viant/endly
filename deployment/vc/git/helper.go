package git

import (
	"github.com/viant/toolbox/url"
	"gopkg.in/src-d/go-git.v4"
)

func matchesOrigin(repository *git.Repository, resource *url.Resource) bool {
	remotes, err := repository.Remotes()
	if err != nil || len(remotes) == 0 {
		return false
	}
	for _, remote := range remotes {
		if len(remote.Config().URLs) == 0 {
			continue
		}
		for _, URL := range remote.Config().URLs {
			if URL == resource.URL {
				return true
			}
			actual := url.NewResource(URL)
			if actual.ParsedURL.Host != resource.ParsedURL.Host {
				continue
			}
			if actual.ParsedURL.Path != resource.ParsedURL.Path {
				continue
			}
			return true
		}
	}
	return false
}
