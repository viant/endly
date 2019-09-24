package storage

import (
	"errors"
	"github.com/viant/endly"
	"github.com/viant/endly/testing/validator"
	"github.com/viant/toolbox/url"
	arl "github.com/viant/afs/url"

)

//ExistsRequest represents exists request
type ExistsRequest struct {
	Source *url.Resource   `required:"true" description:"source asset or directory"`
	Assets []string        `description:"assets locations, joined with source URL"`
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
		Exists:make(map[string]bool),
	}
	err := s.exists(context, request, response)
	return response, err
}

func (s *service) exists(context *endly.Context,  request *ExistsRequest, response *ExistsResponse) error {
	source, storageOpts, err := GetResourceWithOptions(context, request.Source)
	if err != nil {
		return err
	}
	fs, err := StorageService(context, source)
	if err != nil {
		return err
	}
	defer func() {
		_ = fs.Close(source.URL)
	}()
	if len(request.Assets) == 0 {
		exists, err := fs.Exists(context.Background(), source.URL, storageOpts...)
		if err != nil {
			return err
		}
		response.Exists[source.URL] = exists
	} else {
		for i := range request.Assets {
			URL := arl.Join(source.URL, request.Assets[i])
			response.Exists[request.Assets[i]], err = fs.Exists(context.Background(), URL, storageOpts...)
			if err != nil {
				return err
			}
		}
	}
	if request.Expect != nil {
		response.Assert, err = validator.Assert(context, request, request.Expect, response.Exists, "Exists", "assert Exists responses")
	}
	return nil
}


//Init initialises Upload request
func (r *ExistsRequest) Init() error {
	if len(r.Expect) > 0 && len(r.Assets) == 0  {
		r.Assets = make([]string, 0)
		for k:= range r.Expect {
			r.Assets = append(r.Assets, k)
		}
	}


	return nil
}


//Validate checks if request is valid
func (r *ExistsRequest) Validate() error {
	if r.Source == nil {
		return errors.New("source was empty")
	}
	return nil
}
