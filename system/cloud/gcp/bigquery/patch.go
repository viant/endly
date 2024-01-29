package bigquery

import (
	"github.com/pkg/errors"
	"github.com/viant/endly"
	"google.golang.org/api/bigquery/v2"
)

// PatchRequest represents a patch request
type PatchRequest struct {
	*bigquery.TableReference
	Table       string
	TemplateRef *bigquery.TableReference
	Template    string
	Schema      *bigquery.TableSchema
}

type PatchResponse struct {
	*bigquery.Table
}

// Init initialises request
func (r *PatchRequest) Init() (err error) {
	if r.Table != "" && (r.TableReference == nil || r.TableReference.TableId == "") {
		if r.TableReference, err = NewTableReference(r.Table); err != nil {
			return err
		}
	}
	if r.TemplateRef == nil && r.Template != "" {
		if r.TemplateRef, err = NewTableReference(r.Template); err != nil {
			return err
		}
	}
	return err
}

// Validate checks if request is valid
func (r *PatchRequest) Validate() (err error) {
	if r.TableReference == nil {
		return errors.New("table was empty")
	}
	if r.Schema == nil && r.Template == "" {
		return errors.New("schema was empty")
	}
	return nil
}

func (s *service) patch(context *endly.Context, request *PatchRequest) (*PatchResponse, error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	tableService := bigquery.NewTablesService(client.service)
	schema := request.Schema
	if schema == nil {
		table, err := s.table(context, request.TemplateRef)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get table for %v", request.TemplateRef)
		}
		schema = table.Schema
	}

	table, err := s.table(context, request.TableReference)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get table for %v", request.TableId)
	}
	table.Schema = schema
	request.Schema = schema
	call := tableService.Patch(request.ProjectId, request.DatasetId, request.TableId, table)
	call.Context(context.Background())
	response, err := call.Do()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to patch table: %v:%v.%v, %v", request.ProjectId, request.DatasetId, request.TableId, table)
	}
	return &PatchResponse{
		Table: &bigquery.Table{
			TableReference:    response.TableReference,
			Id:                response.Id,
			Description:       response.Description,
			Schema:            response.Schema,
			Clustering:        response.Clustering,
			CreationTime:      response.CreationTime,
			RangePartitioning: response.RangePartitioning,
			NumRows:           response.NumRows,
		},
	}, nil

}
