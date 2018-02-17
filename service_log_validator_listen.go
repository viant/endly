package endly

import (
	"github.com/viant/toolbox/url"
	"regexp"
)

//LogType represents  a log type
type LogType struct {
	Name         string `required:"true" description:"log type name"`
	Format       string `description:"log format"`
	Mask         string `description:"expected log file mast"`
	Exclusion    string `description:"if specified, exclusion fragment can not match log record"`
	Inclusion    string `description:"if specified, inclusion fragment must match log record"`
	IndexRegExpr string `description:"provide expression for indexing log messages, in this case position based logging will not apply"` //provide expression for indexing log message, in this case position based logging will not apply
	indexExpr    *regexp.Regexp
}

//LogValidatorListenRequest represents listen for a logs request.
type LogValidatorListenRequest struct {
	FrequencyMs int
	Source      *url.Resource `required:"true" description:"log location"`
	Types       []*LogType    `required:"true" description:"log types"`
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
