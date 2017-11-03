package endly

import (
	"github.com/viant/toolbox/url"
)

//LogType represents  a log type
type LogType struct {
	Name    string
	Format  string
	Mask    string
	Exclusion string
	Inclusion string
}

//LogValidatorListenRequest represents listen for a logs request.
type LogValidatorListenRequest struct {
	FrequencyMs int
	Source      *url.Resource
	Types       []*LogType
}

//LogValidatorListenResponse represents a log validation listen response.
type LogValidatorListenResponse struct {
	Meta LogTypesMeta
}

//LogTypesMeta represents log type meta details
type LogTypesMeta map[string]*LogTypeMeta
