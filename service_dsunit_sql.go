package endly

import (
	"github.com/viant/toolbox/url"
	"errors"
	"fmt"
)

//DsUnitSQLScriptRequest represents a SQL script request.
type DsUnitSQLScriptRequest struct {
	Datastore string
	Scripts   []*url.Resource
}



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


