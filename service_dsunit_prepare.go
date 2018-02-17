package endly

import (
	"github.com/viant/dsunit"
)

//DsUnitPrepareRequest represents a dsunit prepare requests.
type DsUnitPrepareRequest struct {
	*DsUnitDataRequest
}

//DsUnitPrepareResponse represents dsunit prepare response.
type DsUnitPrepareResponse struct {
	Added    int
	Modified int
	Deleted  int
}

//AsDatasetResource converts request as *dsunit.DatasetResource
func (r *DsUnitPrepareRequest) AsDatasetResource() *dsunit.DatasetResource {
	var result = &dsunit.DatasetResource{
		Datastore:  r.Datastore,
		URL:        r.URL,
		Credential: r.Credential,
		Prefix:     r.Prefix,
		Postfix:    r.Postfix,
	}
	if len(r.Data) > 0 {
		result.TableRows = make([]*dsunit.TableRows, 0)
		for table, data := range r.Data {
			var tableRows = &dsunit.TableRows{
				Table: table,
				Rows:  data,
			}
			result.TableRows = append(result.TableRows, tableRows)
		}
	}
	return result
}
