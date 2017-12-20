package reporter

type Response struct {
	Status string
	Error  string
}

type RegisterReportRequest struct {
	ReportType string
	Report     interface{}
}

type RegisterReportResponse struct {
	*Response
}

type RunReportRequest struct {
	Name       string
	Datastore  string
	Parameters map[string]interface{}
}

type RunReportResponse struct {
	*Response
	Name    string
	Status  string
	Columns []string
	Data    []map[string]interface{}
}
