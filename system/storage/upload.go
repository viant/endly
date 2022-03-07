package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/viant/afs/file"
	"github.com/viant/afs/option"
	"github.com/viant/afs/storage"
	"github.com/viant/endly"
	"github.com/viant/endly/udf"
	"github.com/viant/toolbox/url"
	"io"
	"os"
	"strings"
)

//UploadRequest represents a resources Upload request, it takes context state key to Upload to target destination.
type UploadRequest struct {
	SourceKey string        `required:"true" description:"state key with asset content"`
	Region    string        `description:"cloud storage region"`
	Mode      int           `description:"os.FileMode"`
	Udf       string        `description:"name of udf to transform payload before placing into state map"` //name of udf function that will be used to transform payload
	Dest      *url.Resource `required:"true" description:"destination asset or directory"`                 //target URL with credentials
}

//UploadResponse represents a Upload response
type UploadResponse struct {
	Size        int
	Transformed string
	URL         string
}

//Upload upload content defined by sourceKey to dest
func (s *service) Upload(context *endly.Context, request *UploadRequest) (*UploadResponse, error) {
	var response = &UploadResponse{}
	return response, s.upload(context, request, response)
}

func (s *service) upload(context *endly.Context, request *UploadRequest, response *UploadResponse) error {
	var options = []storage.Option{}
	if request.Region != "" {
		options = append(options, option.NewRegion(request.Region))
	}
	dest, storageOpts, err := GetResourceWithOptions(context, request.Dest, options...)
	if err != nil {
		return err
	}
	fs, err := StorageService(context, dest)
	if err != nil {
		return err
	}
	var state = context.State()
	if !state.Has(request.SourceKey) {
		return fmt.Errorf("sourcekey %v value was empty", request.SourceKey)
	}
	data := state.GetString(request.SourceKey)
	data = context.Expand(data)
	var reader io.Reader = strings.NewReader(data)
	if request.Udf != "" {
		transformed, err := udf.TransformWithUDF(context, request.Udf, request.SourceKey, string(data))
		if err != nil {
			return fmt.Errorf("failed to transform: %v with %v, due to: %w", data, request.Udf, err)
		}
		switch v := transformed.(type) {
		case []byte:
			reader = bytes.NewReader(v)
			response.Transformed = string(v)
		case map[string]interface{}:
			data, _ := json.Marshal(v)
			reader = strings.NewReader(string(data))
			response.Transformed = string(data)
		case string:
			response.Transformed = v
			reader = strings.NewReader(v)
		}
		if reader == nil {
			return fmt.Errorf("invalid tranformation type:%T", transformed)
		}
	}

	err = fs.Upload(context.Background(), dest.URL, os.FileMode(request.Mode), reader, storageOpts...)
	if err != nil {
		return err
	}
	response.Size = len(data)
	response.URL = dest.URL
	return nil

}

//Init initialises Upload request
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
