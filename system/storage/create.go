package storage

import (
	"errors"

	"github.com/viant/endly"
	"github.com/viant/afs/storage"
	"github.com/viant/afs/file"
	"github.com/viant/toolbox/url"
	"io"
	"os"
	"strings"
)

//CreateRequest represents a resources upload request, it takes context state key to upload to target destination.
type CreateRequest struct {
	SourceKey string `required:"true" description:"state key with asset content"`
	Mode      int    `description:"os.FileMode"`
	IsDir     bool  `description:"is directory flag"`
	Dest      *url.Resource `required:"true" description:"destination asset or directory"` //target URL with credentials
}


//CreateResponse represents a upload response
type CreateResponse struct {
	Size int
	URL  string
}


func (s *service) create(context *endly.Context, request *CreateRequest) (*CreateResponse, error) {
	var response = &CreateResponse{}

	resource, service, err := s.getResourceAndService(context, request.Dest)
	if err != nil {
		return nil, err
	}

	var options = make([]storage.Option, 0)
	if ! request.IsDir {
		var state = context.State()
		if state.Has(request.SourceKey) {
			data := state.GetString(request.SourceKey)
			options = append(options, io.Reader(strings.NewReader(data)))
			response.Size = len(data)
		}
	}
	err = service.Create(context.Background(), resource.URL, os.FileMode(request.Mode),request.IsDir, options...)
	if err != nil {
		return nil, err
	}
	response.URL = resource.URL
	return response, nil

}


//Init initialises upload request
func (r *CreateRequest) Init() error {
	if r.Mode == 0 {
		if r.IsDir {
			r.Mode = int(file.DefaultDirOsMode)
		} else {
			r.Mode = int(file.DefaultFileOsMode)
		}
	}
	return nil
}




//Validate checks if request is valid
func (r *CreateRequest) Validate() error {
	if r.Dest == nil {
		return errors.New("dest was empty")
	}
	if r.SourceKey == "" {
		return errors.New("sourceKey was empty")
	}
	return nil
}
