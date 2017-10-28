package endly

//LogValidatorAssertRequest represents a log assert request
type LogValidatorAssertRequest struct {
	LogWaitTimeMs      int
	LogWaitRetryCount  int
	Description        string
	ExpectedLogRecords []*ExpectedLogRecord
}

//ExpectedLogRecord represents an expected log record.
type ExpectedLogRecord struct {
	Tag      string
	TagIndex string
	Type     string
	Records  []map[string]interface{}
}

//LogValidatorAssertResponse represents a log assert response
type LogValidatorAssertResponse struct {
	Description    string
	ValidationInfo []*ValidationInfo
}
