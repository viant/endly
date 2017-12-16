package endly

import (
	"errors"
	"fmt"
	"github.com/viant/toolbox/url"
)

//DsUnitSQLScriptRequest represents a SQL script request.
type DsUnitSQLScriptRequest struct {
	Datastore string
	Scripts   []*url.Resource
}

//Validate checks if request is valid
func (r *DsUnitSQLScriptRequest) Validate() error {
	if r.Datastore == "" {
		return errors.New("Datastore was empty")
	}
	if len(r.Scripts) == 0 {
		return fmt.Errorf("Scripts was empty on %v", r.Datastore)
	}
	return nil
}

//DsUnitSQLScriptResponse represents a SQL script response.
type DsUnitSQLScriptResponse struct {
	Modified int
}
