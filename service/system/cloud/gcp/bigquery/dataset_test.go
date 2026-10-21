package bigquery

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	"github.com/viant/endly/service/system/cloud/gcp"
	"github.com/viant/toolbox"
	"google.golang.org/api/bigquery/v2"
	"google.golang.org/api/option"
)

type datasetGetRoundTripper func(*http.Request) (*http.Response, error)

func (f datasetGetRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func newDatasetGetTestContext(t *testing.T, transport http.RoundTripper) *endly.Context {
	t.Helper()

	googleService, err := bigquery.NewService(context.Background(), option.WithHTTPClient(&http.Client{Transport: transport}))
	require.NoError(t, err)
	client := &CtxClient{AbstractClient: &gcp.AbstractClient{}}
	client.SetContext(context.Background())
	require.NoError(t, client.SetService(googleService))

	ctx := endly.New().NewContext(nil)
	require.NoError(t, ctx.Replace(clientKey, client))
	return ctx
}

func TestDatasetsGetRouteUsesInitializedGoogleCallAndOptions(t *testing.T) {
	var request *http.Request
	ctx := newDatasetGetTestContext(t, datasetGetRoundTripper(func(req *http.Request) (*http.Response, error) {
		request = req
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"kind":"bigquery#dataset","id":"project id:dataset id","location":"US"}`)),
			Request:    req,
		}, nil
	}))

	svc := New().(*service)
	request = nil
	route, err := svc.Route("datasetsGet")
	require.NoError(t, err)
	req, ok := route.RequestProvider().(*DatasetGetRequest)
	require.True(t, ok)
	require.NoError(t, toolbox.DefaultConverter.AssignConverted(req, map[string]interface{}{
		"projectId":   "project id",
		"datasetId":   "dataset id",
		"ifNoneMatch": `"etag-value"`,
		"urlParams": map[string]interface{}{
			"accessPolicyVersion": 3,
			"datasetView":         "FULL",
			"fields":              "id,location",
			"trace":               "one",
		},
	}))
	result := svc.Run(ctx, req)

	require.Equal(t, "ok", result.Status)
	dataset, ok := result.Response.(*bigquery.Dataset)
	require.True(t, ok)
	require.Equal(t, "project id:dataset id", dataset.Id)
	require.NotNil(t, request)
	require.Equal(t, "/bigquery/v2/projects/project%20id/datasets/dataset%20id", request.URL.EscapedPath())
	query := request.URL.Query()
	require.Equal(t, "3", query.Get("accessPolicyVersion"))
	require.Equal(t, "FULL", query.Get("datasetView"))
	require.Equal(t, "id,location", query.Get("fields"))
	require.Equal(t, "one", query.Get("trace"))
	require.Equal(t, `"etag-value"`, request.Header.Get("If-None-Match"))
}

func TestDatasetsGetRoutePropagatesGoogleAPIError(t *testing.T) {
	ctx := newDatasetGetTestContext(t, datasetGetRoundTripper(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"error":{"code":403,"message":"permission denied","errors":[{"reason":"forbidden"}]}}`)),
			Request:    req,
		}, nil
	}))

	result := New().Run(ctx, &DatasetGetRequest{ProjectId: "project-id", DatasetId: "dataset-id"})
	require.Equal(t, "error", result.Status)
	require.Error(t, result.Err)
	require.Contains(t, result.Error, "permission denied")
}
