package storage

import (
	"errors"
	"github.com/viant/afs"
	"github.com/viant/endly"
	"github.com/viant/endly/model/location"
)

// RemoveRequest represents a resources Remove request
type RemoveRequest struct {
	Assets []*location.Resource `required:"true" description:"resources to Remove"`
}

// RemoveResponse represents a resources Remove response, it returns url of all resource that have been removed.
type RemoveResponse struct {
	Removed []string
}

// Remove removes supplied assets
func (s *service) Remove(context *endly.Context, request *RemoveRequest) (*RemoveResponse, error) {
	var response = &RemoveResponse{
		Removed: make([]string, 0),
	}
	return response, s.remove(context, request, response)
}

// Remove removes supplied assets
func (s *service) remove(context *endly.Context, request *RemoveRequest, response *RemoveResponse) error {
	fs, err := StorageService(context, request.Assets...)
	if err != nil {
		return err
	}
	for _, resource := range request.Assets {
		resource, _ = removeResource(context, resource, fs)
		response.Removed = append(response.Removed, resource.URL)
	}
	return nil
}

// removeResource removes supplied resource
func removeResource(context *endly.Context, resource *location.Resource, fs afs.Service) (*location.Resource, error) {
	resource, storageOpts, err := GetResourceWithOptions(context, resource)
	if err != nil {
		return nil, err
	}
	return resource, fs.Delete(context.Background(), resource.URL, storageOpts...)
}

// NewRemoveRequest creates a new Remove request
func NewRemoveRequest(assets ...*location.Resource) *RemoveRequest {
	return &RemoveRequest{
		Assets: assets,
	}
}

// Validate checks if request is valid
func (r *RemoveRequest) Validate() error {
	if len(r.Assets) == 0 {
		return errors.New("assets was empty")
	}
	return nil
}
