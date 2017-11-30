package endly

import (
	"github.com/viant/toolbox/url"
)

//DsUnitSQLScriptRequest represents a SQL script request.
type DsUnitSQLScriptRequest struct {
	Datastore string
	Scripts   []*url.Resource
}

//DsUnitSQLScriptResponse represents a SQL script response.
type DsUnitSQLScriptResponse struct {
	Modified int
}
