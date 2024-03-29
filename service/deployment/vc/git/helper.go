package git

import (
	"github.com/viant/endly/model/location"
	"gopkg.in/src-d/go-git.v4"
)

func matchesOrigin(repository *git.Repository, resource *location.Resource) bool {
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
			actual := location.NewResource(URL)
			if actual.Hostname() != resource.Hostname() {
				continue
			}
			if actual.Path() != resource.Path() {
				continue
			}
			return true
		}
	}
	return false
}
