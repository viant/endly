package storage

import (
	"github.com/pkg/errors"
	"github.com/viant/endly"
	"github.com/viant/endly/internal/udf"
	"github.com/viant/endly/internal/util"
	"github.com/viant/endly/model/location"
	"github.com/viant/endly/testing/validator"
	"github.com/viant/toolbox"
	"io/ioutil"
)

// DownloadRequest represents a resources Download request, it downloads source into context.state target key
type DownloadRequest struct {
	Source  *location.Resource `required:"true" description:"source asset or directory"`
	DestKey string             `required:"true" description:"state map key destination"`
	Udf     string             `description:"name of udf to transform payload before placing into state map"` //name of udf function that will be used to transform payload
	Expect  interface{}        `description:"if specified expected file content used for validation"`
}

// DownloadResponse represents a Download response
type DownloadResponse struct {
	Info        toolbox.FileInfo
	Payload     string //source content, if binary then is will be prefixed base64: followed by based 64 encoded content.
	Transformed interface{}
	Assert      *validator.AssertResponse
}

func (s *service) Download(context *endly.Context, request *DownloadRequest) (*DownloadResponse, error) {
	var response = &DownloadResponse{}
	return response, s.download(context, request, response)
}

func (s *service) download(context *endly.Context, request *DownloadRequest, response *DownloadResponse) error {
	source, storageOpts, err := GetResourceWithOptions(context, request.Source)
	if err != nil {
		return err
	}
	fs, err := StorageService(context, source)
	if err != nil {
		return err
	}
	reader, err := fs.OpenURL(context.Background(), source.URL, storageOpts...)
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return errors.Wrapf(err, "unable to read %v", source.URL)
	}
	if request.Udf != "" {
		response.Transformed, err = udf.TransformWithUDF(context, request.Udf, source.URL, data)
		if err != nil {
			return errors.Wrapf(err, "failed to transform with %v udf, source: %v", request.Udf, source.URL)
		}
	}
	response.Payload = util.AsPayload(data)
	var payload interface{}
	if response.Transformed != nil {
		payload = response.Transformed
	} else {
		payload = response.Payload
	}
	if request.DestKey != "" {
		var state = context.State()
		state.Put(request.DestKey, payload)
	}
	if request.Expect != nil {
		response.Assert, err = validator.Assert(context, request, request.Expect, payload, "Download.Payload", "assert Download responses")
	}
	return err
}

// Validate checks if request is valid
func (r *DownloadRequest) Validate() error {
	if r.Source == nil {
		return errors.New("source was empty")
	}
	return nil
}
