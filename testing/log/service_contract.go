package log

import (
	"github.com/viant/assertly"
	"github.com/viant/toolbox/url"
	"regexp"
)

//AssertRequest represents a log assert request
type AssertRequest struct {
	LogWaitTimeMs      int
	LogWaitRetryCount  int
	Description        string
	ExpectedLogRecords []*ExpectedRecord `required:"true" description:"expected log data"`
}

//ExpectedRecord represents an expected log record.
type ExpectedRecord struct {
	TagID   string `description:"neatly tag id for matching validation summary"`
	Type    string `required:"true" description:"log type register with listener"`
	Records []interface{}
}

//AssertResponse represents a log assert response
type AssertResponse struct {
	Validations []*assertly.Validation
}

//Assertion returns description with validation slice
func (r *AssertResponse) Assertion() []*assertly.Validation {
	return r.Validations
}

//Type represents  a log type
type Type struct {
	Name         string `required:"true" description:"log type name"`
	Format       string `description:"log format"`
	Mask         string `description:"expected log file mast"`
	Exclusion    string `description:"if specified, exclusion fragment can not match log record"`
	Inclusion    string `description:"if specified, inclusion fragment must match log record"`
	IndexRegExpr string `description:"provide expression for indexing log messages, in this case position based logging will not apply"` //provide expression for indexing log message, in this case position based logging will not apply
	indexExpr    *regexp.Regexp
	UDF          string `description:"registered user defined function to transform content file before applying validation i,e decompress"`
}

//ListenRequest represents listen for a logs request.
type ListenRequest struct {
	FrequencyMs int
	Source      *url.Resource `required:"true" description:"log location"`
	Types       []*Type       `required:"true" description:"log types"`
}

//ListenResponse represents a log validation listen response.
type ListenResponse struct {
	Meta TypesMeta
}

//UseIndex returns true if index can be used.
func (t *Type) UseIndex() bool {
	return t.IndexRegExpr != ""
}

//GetIndexExpr returns index expression.
func (t *Type) GetIndexExpr() (*regexp.Regexp, error) {
	if t.indexExpr != nil {
		return t.indexExpr, nil
	}
	var err error
	t.indexExpr, err = regexp.Compile(t.IndexRegExpr)
	return t.indexExpr, err
}

//ResetRequest represents a log reset request
type ResetRequest struct {
	LogTypes []string `required:"true" description:"log types to reset"`
}

//ResetResponse represents a log reset response
type ResetResponse struct {
	LogFiles []string
}
