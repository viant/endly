package storage

import (
	"errors"
	"github.com/viant/endly"
	"github.com/viant/toolbox/url"
)

//RemoveRequest represents a resources Remove request
type RemoveRequest struct {
	Assets []*url.Resource `required:"true" description:"resources to remove"`
}

//RemoveResponse represents a resources Remove response, it returns url of all resource that have been removed.
type RemoveResponse struct {
	Removed []string
}

func (s *service) remove(context *endly.Context, request *RemoveRequest) (*RemoveResponse, error) {
	var response = &RemoveResponse{
		Removed: make([]string, 0),
	}
	for _, resource := range request.Assets {
		resource, service, err := s.getResourceAndService(context, resource)
		if err != nil {
			return nil, err
		}
		if ok, _ := service.Exists(context.Background(), resource.URL); ! ok {
			continue
		}

		err = service.Delete(context.Background(), resource.URL)
		if err != nil {
			return nil, err
		}
		response.Removed = append(response.Removed, resource.URL)
	}
	return response, nil
}


//NewRemoveRequest creates a new remove request
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
