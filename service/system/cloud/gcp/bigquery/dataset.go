package bigquery

import (
	"fmt"

	"github.com/viant/endly"
	"github.com/viant/endly/service/system/cloud/gcp"
	"github.com/viant/toolbox"
	"google.golang.org/api/bigquery/v2"
	"google.golang.org/api/googleapi"
)

// DatasetGetRequest identifies the BigQuery dataset to retrieve. URLParams
// accepts the same optional query parameters supported by Datasets.Get.
type DatasetGetRequest struct {
	ProjectId   string
	DatasetId   string
	URLParams   map[string]interface{}
	IfNoneMatch string
}

func datasetGetRoute(s *service) *endly.Route {
	return &endly.Route{
		Action: "datasetsGet",
		RequestInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T.%v(%T)", s, "datasetsGet", &DatasetGetRequest{}),
		},
		ResponseInfo: &endly.ActionInfo{
			Description: fmt.Sprintf("%T", &bigquery.Dataset{}),
		},
		RequestProvider:  func() interface{} { return &DatasetGetRequest{} },
		ResponseProvider: func() interface{} { return &bigquery.Dataset{} },
		OnRawRequest:     InitRequest,
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			req, ok := request.(*DatasetGetRequest)
			if !ok {
				return nil, fmt.Errorf("unsupported request type: %T", request)
			}
			output, err := s.datasetGet(context, req)
			if err != nil {
				return nil, err
			}
			if context.IsLoggingEnabled() {
				context.Publish(gcp.NewOutputEvent("Get", "proxy", output))
			}
			return output, nil
		},
	}
}

func (s *service) datasetGet(context *endly.Context, request *DatasetGetRequest) (*bigquery.Dataset, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	call := client.service.Datasets.Get(request.ProjectId, request.DatasetId).Context(client.Context())
	if request.IfNoneMatch != "" {
		call.IfNoneMatch(request.IfNoneMatch)
	}

	// Preserve the generic GCP adapter's query conversion while constructing
	// the call through Google's public API, which initializes its service.
	var options []googleapi.CallOption
	for name, value := range request.URLParams {
		options = append(options, googleapi.QueryParameter(name, toolbox.AsString(value)))
	}
	return call.Do(options...)
}
