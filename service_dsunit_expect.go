package endly

import (
	"github.com/viant/assertly"
	"github.com/viant/dsunit"
)

//DsUnitExpectRequest represent verification request.
type DsUnitExpectRequest struct {
	*DsUnitDataRequest
	CheckPolicy int `description:"verification policy"`
}

//AsDatasetResource converts request as *dsunit.DatasetResource
func (r *DsUnitExpectRequest) AsDatasetResource() *dsunit.DatasetResource {
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

//DsUnitExpectResponse represent dsunit expect response
type DsUnitExpectResponse struct {
	*assertly.Validation
}
