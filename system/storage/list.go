package storage

import (
	"github.com/pkg/errors"
	"github.com/viant/afs/asset"
	"github.com/viant/afs/storage"
	"github.com/viant/endly"
	"github.com/viant/endly/system/storage/copy"
	"github.com/viant/endly/testing/validator"
	"github.com/viant/toolbox/url"
	"io/ioutil"
)

//DownloadRequest represents a resources Download request, it downloads source into context.state target key
type ListRequest struct {
	Source         *url.Resource `required:"true" description:"source asset or directory"`
	Matcher        *copy.Matcher
	IncludeContent bool
	Expect         interface{}
}

//DownloadResponse represents a Download response
type ListResponse struct {
	Assets []*asset.Resource
	Assert *validator.AssertResponse
}

//Remove removes supplied assets
func (s *service) List(context *endly.Context, request *ListRequest) (*ListResponse, error) {
	var response = &ListResponse{
		Assets: make([]*asset.Resource, 0),
	}
	return response, s.list(context, request, response)
}

func (s *service) list(context *endly.Context, request *ListRequest, response *ListResponse) error {
	options, err := getMatcherOptions(request)
	if err != nil {
		return err
	}
	source, storageOpts, err := GetResourceWithOptions(context, request.Source, options...)
	if err != nil {
		return err
	}
	fs, err := StorageService(context, source)
	if err != nil {
		return err
	}
	objects, err := fs.List(context.Background(), source.URL, storageOpts...)
	if err != nil {
		return err
	}
	defer func() {
		_ = fs.Close(source.URL)
	}()
	for _, object := range objects {
		var resource *asset.Resource
		if object.IsDir() {
			resource = asset.NewDir(object.URL(), object.Mode())
		} else {
			resource = asset.NewFile(object.URL(), nil, object.Mode())
		}
		if request.IncludeContent && !object.IsDir() {
			reader, err := fs.Download(context.Background(), object)
			if err != nil {
				return errors.Wrapf(err, "failed to download listed content %v", object.URL())
			}
			resource.Data, err = ioutil.ReadAll(reader)
			_ = reader.Close()
			if err != nil {
				return errors.Wrapf(err, "failed to read listed content %v", object.URL())
			}
		}
		response.Assets = append(response.Assets, resource)
	}
	if request.Expect != nil {
		response.Assert, err = validator.Assert(context, request, request.Expect, response.Assets, "List", "assert list responses")
	}
	return nil
}

func getMatcherOptions(request *ListRequest) ([]storage.Option, error) {
	var options = make([]storage.Option, 0)
	if request.Matcher != nil {
		matcher, err := request.Matcher.Matcher()
		if err != nil {
			return nil, err
		}
		options = append(options, matcher)
	}
	return options, nil
}

//Validate checks if request is valid
func (r *ListRequest) Validate() error {
	if r.Source == nil {
		return errors.New("source was empty")
	}
	return nil
}
