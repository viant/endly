package endly

//LogValidatorResetRequest represents a log reset request
type LogValidatorResetRequest struct {
	LogTypes []string
}

//LogValidatorResetResponse represents a log reset response
type LogValidatorResetResponse struct {
	LogFiles []string
}
