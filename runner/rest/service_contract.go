package reset

//SendRequest represents a send request
type SendRequest struct {
	URL     string
	Method  string
	Request interface{}
}

//SendResponse represents a rest response
type SendResponse struct {
	Response interface{}
}
