package storage

import (
	"errors"
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/afs/file"
	"github.com/viant/toolbox/url"
	"os"
	"strings"
)

//UploadRequest represents a resources upload request, it takes context state key to upload to target destination.
type UploadRequest struct {
	SourceKey string `required:"true" description:"state key with asset content"`
	Mode      int `description:"os.FileMode"`
	Dest      *url.Resource `required:"true" description:"destination asset or directory"` //target URL with credentials
}

//UploadResponse represents a upload response
type UploadResponse struct {
	UploadSize int
	UploadURL  string
}


func (s *service) upload(context *endly.Context, request *UploadRequest) (*UploadResponse, error) {
	var response = &UploadResponse{}

	resource, service, err := s.getResourceAndService(context, request.Dest)
	if err != nil {
		return nil, err
	}

	var state = context.State()
	if !state.Has(request.SourceKey) {
		return nil, fmt.Errorf("sourcekey %v value was empty", request.SourceKey)

	}

	data := state.GetString(request.SourceKey)
	err = service.Upload(context.Background(), resource.URL, os.FileMode(request.Mode), strings.NewReader(data))
	if err != nil {
		return nil, err
	}
	response.UploadSize = len(data)
	response.UploadURL = resource.URL
	return response, nil

}

//Init initialises upload request
func (r *UploadRequest) Init() error {
	if r.Mode == 0 {
		r.Mode = int(file.DefaultFileOsMode)
	}
	return nil
}


//Validate checks if request is valid
func (r *UploadRequest) Validate() error {
	if r.Dest == nil {
		return errors.New("dest was empty")
	}
	if r.SourceKey == "" {
		return errors.New("sourceKey was empty")
	}
	return nil
}
