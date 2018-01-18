package endly

import (
	"github.com/viant/toolbox/url"
	"regexp"
)

//LogType represents  a log type
type LogType struct {
	Name         string
	Format       string
	Mask         string
	Exclusion    string
	Inclusion    string
	IndexRegExpr string //provide expression for indexing log message, in this case position based logging will not apply
	indexExpr    *regexp.Regexp
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

//UseIndex returns true if index can be used.
func (t *LogType) UseIndex() bool {
	return t.IndexRegExpr != ""
}

//GetIndexExpr returns index expression.
func (t *LogType) GetIndexExpr() (*regexp.Regexp, error) {
	if t.indexExpr != nil {
		return t.indexExpr, nil
	}
	var err error
	t.indexExpr, err = regexp.Compile(t.IndexRegExpr)
	return t.indexExpr, err
}
