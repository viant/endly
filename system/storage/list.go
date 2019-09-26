package storage

import (
	"context"
	"github.com/pkg/errors"
	"github.com/viant/afs/asset"
	"github.com/viant/afs/matcher"
	"github.com/viant/afs/storage"
	"github.com/viant/endly"
	"github.com/viant/endly/system/storage/copy"
	"github.com/viant/endly/testing/validator"
	"github.com/viant/toolbox/url"
	"io/ioutil"
)

//DownloadRequest represents a resources Download request, it downloads source into context.state target key
type ListRequest struct {
	Source    *url.Resource `required:"true" description:"source asset or directory"`
	Match     *copy.Matcher
	Content   bool
	Recursive bool
	Expect    interface{}
}

//DownloadResponse represents a Download response
type ListResponse struct {
	URL    string
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
	response.URL = source.URL
	fs, err := StorageService(context, source)
	if err != nil {
		return err
	}
	defer func() {
		_ = fs.Close(source.URL)
	}()
	if err = listResource(context.Background(), source.URL, storageOpts, request, response); err != nil {
		return err
	}
	if request.Expect != nil {
		response.Assert, err = validator.Assert(context, request, request.Expect, response.Assets, "List", "assert list responses")
	}
	return nil
}

func listResource(ctx context.Context, URL string, storageOptions []storage.Option, request *ListRequest, response *ListResponse) error {
	objects, err := fs.List(context.Background(), URL, storageOptions...)
	if err != nil {
		return err
	}

	for i, object := range objects {
		var resource *asset.Resource
		if object.IsDir() {
			if i == 0 {
				continue
			}
			resource = asset.NewDir(object.URL(), object.Mode())
		} else {
			resource = asset.NewFile(object.URL(), nil, object.Mode())
		}
		if request.Content && !object.IsDir() {
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

	if request.Recursive {
		dirMatcher := &matcher.Basic{Directory: &request.Recursive}
		objects, err := fs.List(ctx, URL, dirMatcher.Match)
		if err != nil {
			return err
		}

		for i, object := range objects {
			if i == 0 {
				continue
			}
			if err = listResource(context.Background(), object.URL(), storageOptions, request, response); err != nil {
				return err
			}

		}
	}

	return nil
}

func getMatcherOptions(request *ListRequest) ([]storage.Option, error) {
	var options = make([]storage.Option, 0)
	if request.Match != nil {
		matcher, err := request.Match.Matcher()
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
