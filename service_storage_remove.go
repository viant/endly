package endly

import (
	"errors"
	"github.com/viant/toolbox/url"
)

//StorageRemoveRequest represents a resources Remove request
type StorageRemoveRequest struct {
	Resources []*url.Resource `required:"true" description:"resources to remove"`
}

//Validate checks if request is valid
func (r *StorageRemoveRequest) Validate() error {
	if len(r.Resources) == 0 {
		return errors.New("resources was empty")
	}
	return nil
}

//StorageRemoveResponse represents a resources Remove response, it returns url of all resource that have been removed.
type StorageRemoveResponse struct {
	Removed []string
}
