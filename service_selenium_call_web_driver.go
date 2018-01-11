package endly

//SeleniumWebDriverCallRequest represents selenium call driver request
type SeleniumWebDriverCallRequest struct {
	SessionID string
	Call      *SeleniumMethodCall
}

//SeleniumServiceCallResponse represents selenium call response
type SeleniumServiceCallResponse struct {
	Result    []interface{}
	Extracted map[string]string
}
