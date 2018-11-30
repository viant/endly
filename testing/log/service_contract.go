package log

import (
	"fmt"
	"github.com/viant/assertly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
	"regexp"
)

//AssertRequest represents a log assert request
type AssertRequest struct {
	LogWaitTimeMs     int
	LogWaitRetryCount int
	Description       string
	Expect            []*ExpectedRecord `required:"true" description:"expected log data"`
}

//Init converts yaml kv pairs to a map if applicable
func (r *AssertRequest) Init() error {
	if len(r.Expect) == 0 {
		return nil
	}
	for _, expecRecords := range r.Expect {
		if len(expecRecords.Records) == 0 {
			continue
		}
		for i, record := range expecRecords.Records {
			if toolbox.IsSlice(record) {
				if aMap, err := toolbox.ToMap(record); err == nil {
					expecRecords.Records[i] = aMap
				}
			}
		}
	}
	return nil
}

//Validate check if request is valid
func (r *AssertRequest) Validate() error {
	if len(r.Expect) == 0 {
		return nil
	}
	for i, expecRecords := range r.Expect {
		if expecRecords.Type == "" {
			return fmt.Errorf("Expect[%d].Type was empty", i)
		}
	}
	return nil
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
	UDF          string `description:"registered user defined function to transform content file before applying validation"`
	Debug        bool	`description:"if set, every record appended to validation queue will be listed"`
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
