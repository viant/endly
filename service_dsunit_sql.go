package endly

import (
	"errors"
	"fmt"
	"github.com/viant/toolbox/url"
)

//DsUnitSQLRequest represents a SQL script request.
type DsUnitSQLRequest struct {
	Datastore string
	Scripts   []*url.Resource
	SQLs      []string
}

//Validate checks if request is valid
func (r *DsUnitSQLRequest) Validate() error {
	if r.Datastore == "" {
		return errors.New("Datastore was empty")
	}
	if len(r.Scripts) == 0 && len(r.SQLs) == 0 {
		return fmt.Errorf("Scripts/SQLs were empty on %v", r.Datastore)
	}
	return nil
}

//DsUnitSQLScriptResponse represents a SQL script response.
type DsUnitSQLScriptResponse struct {
	Modified int
}
