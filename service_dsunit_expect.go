package endly

import (
	"fmt"
	"github.com/viant/dsunit"
)

//DsUnitExpectRequest represent verification request.
type DsUnitExpectRequest struct {
	Datasets *dsunit.DatasetResource
	//table to table rows data
	Data        map[string][]map[string]interface{}
	Expand      bool
	CheckPolicy int
}

//Validate checks if request if valid, otherwise returns an error.
func (r *DsUnitExpectRequest) Validate() error {
	if len(r.Data) > 0 && r.Datasets != nil {
		r.Datasets.TableRows = make([]*dsunit.TableRows, 0)
		for table, data := range r.Data {
			var tableRows = &dsunit.TableRows{
				Table: table,
				Rows:  data,
			}
			r.Datasets.TableRows = append(r.Datasets.TableRows, tableRows)
		}
	}
	if r.Datasets == nil {
		return fmt.Errorf("Datasets was nil")
	}
	if r.Datasets.Datastore == "" {
		return fmt.Errorf("Datasets.Datastore was empty")
	}

	if r.Datasets.URL == "" && len(r.Datasets.TableRows) == 0 {
		return fmt.Errorf("Missing data: Datasets.URL/Datasets.TableRows were empty")
	}
	return nil
}

//DsUnitExpectResponse represents verification response
type DsUnitExpectResponse struct {
	DatasetChecked map[string]int
}
