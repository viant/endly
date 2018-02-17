package endly

import "github.com/viant/assertly"

//LogValidatorAssertRequest represents a log assert request
type LogValidatorAssertRequest struct {
	LogWaitTimeMs      int
	LogWaitRetryCount  int
	Description        string
	ExpectedLogRecords []*ExpectedLogRecord `required:"true" description:"expected log data"`
}

//ExpectedLogRecord represents an expected log record.
type ExpectedLogRecord struct {
	TagID   string `description:"neatly tag id for matching validation summary"`
	Type    string `required:"true" description:"log type register with listener"`
	Records []interface{}
}

//LogValidatorAssertResponse represents a log assert response
type LogValidatorAssertResponse struct {
	Description string
	Validations []*assertly.Validation
}
