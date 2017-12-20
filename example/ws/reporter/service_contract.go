package reporter

//Response represents generic response
type Response struct {
	Status string
	Error  string
}

//RegisterReportRequest represents register request
type RegisterReportRequest struct {
	ReportType string
	Report     interface{}
}

//RegisterReportRequest represents register response
type RegisterReportResponse struct {
	*Response
}

//RunReportRequest represents run request
type RunReportRequest struct {
	Name       string
	Datastore  string
	Parameters map[string]interface{}
}

//RunReportRequest represents run response
type RunReportResponse struct {
	*Response
	Name    string
	Status  string
	Columns []string
	Data    []map[string]interface{}
}
