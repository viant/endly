package dsunit

import (
	"fmt"
	"regexp"

	db "github.com/viant/dsunit"
	dburl "github.com/viant/dsunit/url"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
)

// CollectExpectedRequest freezes one case's expected values after its sequence
// hydration. Collection never reads or modifies the datastore.
type CollectExpectedRequest struct {
	Bucket    string
	Datastore string
	URL       string
	Data      map[string][]map[string]interface{}
}
type CollectExpectedResponse struct {
	Tables int
	Rows   int
}

func collectExpected(ctx *endly.Context, req *CollectExpectedRequest) (*CollectExpectedResponse, error) {
	if !regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.]*$`).MatchString(req.Bucket) || req.Datastore == "" {
		return nil, fmt.Errorf("expected batch bucket and datastore are required")
	}
	resource := &db.DatasetResource{Resource: &dburl.Resource{URL: req.URL}, DatastoreDatasets: &db.DatastoreDatasets{Datastore: req.Datastore, Data: req.Data}}
	if err := resource.Load(); err != nil {
		return nil, err
	}
	state := ctx.State()
	key := req.Bucket + "." + req.Datastore
	combined := map[string][]map[string]interface{}{}
	if previous, ok := state.GetValue(key); ok {
		if err := toolbox.DefaultConverter.AssignConverted(&combined, previous); err != nil {
			return nil, err
		}
	}
	response := &CollectExpectedResponse{}
	// Build a replacement map before publication so an invalid case cannot
	// partially alter the pending group expectation batch.
	next := map[string][]map[string]interface{}{}
	for table, rows := range combined {
		next[table] = append([]map[string]interface{}{}, rows...)
	}
	for _, dataset := range resource.Datasets {
		for _, row := range dataset.Records {
			expanded := toolbox.AsMap(state.Expand(row))
			next[dataset.Table] = append(next[dataset.Table], expanded)
			response.Rows++
		}
		response.Tables++
	}
	if response.Rows == 0 {
		return nil, fmt.Errorf("case has no expected database rows")
	}
	state.SetValue(key, next)
	return response, nil
}
func (s *service) registerExpectedBatchRoute() {
	s.Register(&endly.Route{Action: "collectExpected", RequestInfo: &endly.ActionInfo{Description: "Freeze case-owned expected database values for a later group dsunit expect batch; no SQL executes"}, RequestProvider: func() interface{} { return &CollectExpectedRequest{} }, ResponseProvider: func() interface{} { return &CollectExpectedResponse{} }, Handler: func(ctx *endly.Context, request interface{}) (interface{}, error) {
		return collectExpected(ctx, request.(*CollectExpectedRequest))
	}})
}
