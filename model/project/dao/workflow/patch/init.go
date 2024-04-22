package patch

import (
	"context"
	"github.com/viant/datly"
	"github.com/viant/datly/repository"
	"github.com/viant/datly/repository/contract"
	"reflect"
)

var WorkflowPatchPathURI = "/v1/api/endly/workflow/{ProjectID}"

func DefineWorkflowPatchComponent(ctx context.Context, srv *datly.Service) (*repository.Component, error) {
	return srv.AddHandler(ctx, contract.NewPath("PATCH", WorkflowPatchPathURI), &Handler{},
		repository.WithContract(
			reflect.TypeOf(&Input{}),
			reflect.TypeOf(&Output{}),
			&WorkflowPatchFS))
}
