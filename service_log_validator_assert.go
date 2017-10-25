package endly

//LogValidatorAssertRequest represents a log assert request
type LogValidatorAssertRequest struct {
	LogWaitTimeMs      int
	LogWaitRetryCount  int
	ExpectedLogRecords []*ExpectedLogRecord
}

//ExpectedLogRecord represents an expected log record.
type ExpectedLogRecord struct {
	Tag     string
	Type    string
	Records []map[string]interface{}
}
