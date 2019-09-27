package storage

import (
	"errors"
	"github.com/viant/endly"
	"github.com/viant/endly/testing/validator"
	"github.com/viant/toolbox/url"
)

//ExistsRequest represents exists request
type ExistsRequest struct {
	Assets []*url.Resource `required:"true" description:"source asset or directory"`
	Expect map[string]bool `description:"map of asset and exists flag"`
}

//ExistsResponse represents exists response
type ExistsResponse struct {
	Exists map[string]bool `description:"locations with exists flag, when assets uses only asset location otherwise source URL"`
	Assert *validator.AssertResponse
}

//Exists checks if supplied asset exists
func (s *service) Exists(context *endly.Context, request *ExistsRequest) (*ExistsResponse, error) {
	var response = &ExistsResponse{
		Exists: make(map[string]bool),
	}
	err := s.exists(context, request, response)
	return response, err
}

func (s *service) exists(context *endly.Context, request *ExistsRequest, response *ExistsResponse) error {
	fs, err := StorageService(context, request.Assets...)
	if err != nil {
		return err
	}
	for _, asset := range request.Assets {
		URL := context.Expand(asset.URL)
		source, storageOpts, err := GetResourceWithOptions(context, asset)
		if err != nil {
			return err
		}
		exists, err := fs.Exists(context.Background(), source.URL, storageOpts...)
		if err != nil {
			return err
		}
		response.Exists[URL] = exists
	}
	if request.Expect != nil {
		response.Assert, err = validator.Assert(context, request, request.Expect, response.Exists, "Exists", "assert Exists responses")
	}
	return nil
}

//Validate checks if request is valid
func (r *ExistsRequest) Validate() error {
	if r.Assets == nil {
		return errors.New("assets was empty")
	}
	return nil
}
