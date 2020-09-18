package bigquery

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/system/storage"
	"github.com/viant/toolbox/url"
	"google.golang.org/api/bigquery/v2"
)

const tableKey = "bigqery_table"

//TableRequest table request
type TableRequest struct {
	Table string
	*bigquery.TableReference
	Dest *url.Resource
}

func (r *TableRequest) Init() (err error) {
	if r.TableReference == nil && r.Table != "" {
		r.TableReference, err = NewTableReference(r.Table)
	}
	return err
}

func (r *TableRequest) Validate() (err error) {
	if r.TableReference == nil {
		return errors.New("table was empty")
	}
	return nil
}

//TableResponse table response
type TableResponse struct {
	Table *bigquery.Table
}

//Table returns a table
func (s *service) Table(context *endly.Context, request *TableRequest) (*TableResponse, error) {
	table, err := s.table(context, request.TableReference)
	if err != nil {
		return nil, err
	}
	response := &TableResponse{
		Table: table,
	}
	if request.Dest != nil {
		dest, err := context.ExpandResource(request.Dest)
		if err != nil {
			return nil, err
		}
		data, err := json.Marshal(table)
		if err != nil {
			return nil, err
		}
		key := fmt.Sprintf("%v_%v", tableKey+request.TableId)
		state := context.State()
		state.Put(key, data)
		if err = endly.Run(context, &storage.UploadRequest{SourceKey: key, Dest: dest}, nil); err != nil {
			return nil, err
		}
	}
	return response, nil
}

//Table returns bif query table
func (s *service) table(context *endly.Context, reference *bigquery.TableReference) (table *bigquery.Table, err error) {
	client, err := GetClient(context)
	if err != nil {
		return nil, err
	}
	call := bigquery.NewTablesService(client.service).Get(reference.ProjectId, reference.DatasetId, reference.TableId)
	call.Context(context.Background())
	if table, err = call.Do(); err == nil {
		return table, err
	}
	return table, err
}
