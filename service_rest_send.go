package endly

//RestSendRequest represents a send request
type RestSendRequest struct {
	URL     string
	Method  string
	Request interface{}
}

//RestSendResponse represents a rest response
type RestSendResponse struct {
	Response interface{}
}
