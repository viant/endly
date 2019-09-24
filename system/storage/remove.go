package storage

import (
	"errors"
	"github.com/viant/endly"
	"github.com/viant/afs"
	arl "github.com/viant/afs/url"
	"github.com/viant/afs/file"
	"github.com/viant/toolbox/url"
)

//RemoveRequest represents a resources Remove request
type RemoveRequest struct {
	Assets []*url.Resource `required:"true" description:"resources to Remove"`
}

//RemoveResponse represents a resources Remove response, it returns url of all resource that have been removed.
type RemoveResponse struct {
	Removed []string
}

//Remove removes supplied assets
func (s *service) Remove(context *endly.Context, request *RemoveRequest) (*RemoveResponse, error) {
	var response = &RemoveResponse{
		Removed: make([]string, 0),
	}
	return response, s.remove(context, request, response)
}

//Remove removes supplied assets
func (s *service) remove(context *endly.Context, request *RemoveRequest, response *RemoveResponse) error {
	fs, err := StorageService(context, request.Assets...)
	if err != nil {
		return err
	}
	var baseURLs = make(map[string]bool)
	for _, resource := range request.Assets {
		resource, err = removeResource(context, resource, fs)
		if err != nil {
			return err
		}
		response.Removed = append(response.Removed, resource.URL)
		baseURL, _ := arl.Base(resource.URL, file.Scheme)
		baseURLs[baseURL] = true
	}

	for baseURL := range baseURLs {
		_ = fs.Close(baseURL)
	}
	return nil
}

//removeResource removes supplied resource
func removeResource(context *endly.Context, resource *url.Resource, fs afs.Service) (*url.Resource, error) {
	resource, storageOpts, err := GetResourceWithOptions(context, resource)
	if err != nil {
		return nil, err
	}
	return resource, fs.Delete(context.Background(), resource.URL, storageOpts...)
}


//NewRemoveRequest creates a new Remove request
func NewRemoveRequest(assets ...*url.Resource) *RemoveRequest {
	return &RemoveRequest{
		Assets: assets,
	}
}

//Validate checks if request is valid
func (r *RemoveRequest) Validate() error {
	if len(r.Assets) == 0 {
		return errors.New("assets was empty")
	}
	return nil
}
