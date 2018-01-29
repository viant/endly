package endly

import "github.com/viant/assertly"

//LogValidatorAssertRequest represents a log assert request
type LogValidatorAssertRequest struct {
	LogWaitTimeMs      int
	LogWaitRetryCount  int
	Description        string
	ExpectedLogRecords []*ExpectedLogRecord
}

//ExpectedLogRecord represents an expected log record.
type ExpectedLogRecord struct {
	TagID   string
	Type    string
	Records []interface{}
}

//LogValidatorAssertResponse represents a log assert response
type LogValidatorAssertResponse struct {
	Description string
	Validations []*assertly.Validation
}
