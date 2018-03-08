package reset

//Request represents a send request
type Request struct {
	URL     string
	Method  string
	Request interface{}
}

//Response represents a rest response
type Response struct {
	Response interface{}
}
