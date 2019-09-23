package storage

import (
	"errors"
	"github.com/viant/endly"
	"github.com/viant/endly/testing/validator"
	"github.com/viant/endly/udf"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"io/ioutil"
)

//DownloadRequest represents a resources download request, it downloads source into context.state target key
type DownloadRequest struct {
	Source  *url.Resource `required:"true" description:"source asset or directory"`
	DestKey string        `required:"true" description:"state map key destination"`
	Udf     string        `description:"name of udf to transform payload before placing into state map"` //name of udf function that will be used to transform payload
	Expect  interface{}   `description:"if specified expected file content used for validation"`
}


//DownloadResponse represents a download response
type DownloadResponse struct {
	Info        toolbox.FileInfo
	Payload     string //source content, if binary then is will be prefixed base64: followed by based 64 encoded content.
	Transformed interface{}
	Assert      *validator.AssertResponse
}


func (s *service) download(context *endly.Context, request *DownloadRequest) (*DownloadResponse, error) {
	var response = &DownloadResponse{}
	resource, service, err := s.getResourceAndService(context, request.Source)
	if err != nil {
		return nil, err
	}

	reader, err := service.DownloadWithURL(context.Background(), resource.URL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if request.Udf != "" {
		response.Transformed, err = udf.TransformWithUDF(context, request.Udf, resource.URL, data)
		if err != nil {
			return nil, err
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
		response.Assert, err = validator.Assert(context, request, request.Expect, payload, "Download.Payload", "assert download responses")
	}
	return response, nil

}



//Validate checks if request is valid
func (r *DownloadRequest) Validate() error {
	if r.Source == nil {
		return errors.New("source was empty")
	}
	return nil
}
