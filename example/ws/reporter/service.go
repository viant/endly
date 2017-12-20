package reporter

import (
	"fmt"
	"github.com/viant/dsc"
	"github.com/viant/toolbox"
	"strings"
)

type Service interface {
	Register(request *RegisterReportRequest) *RegisterReportResponse

	Run(request *RunReportRequest) *RunReportResponse

	Datastores() DatastoreRegistry

	ReportProviders() ReportProviders
}

type service struct {
	datastores DatastoreRegistry
	providers  ReportProviders
	reports    Reports
	reportDao  *reportDao
	config     *Config
}

func setError(response *Response, errorMessage string) {
	response.Status = "error"
	response.Error = errorMessage

}

func (s *service) Register(request *RegisterReportRequest) *RegisterReportResponse {
	var response = &RegisterReportResponse{
		Response: &Response{
			Status: "ok",
		},
	}
	var provider, ok = s.providers[request.ReportType]
	if !ok {
		setError(response.Response, fmt.Sprint("failed to lookup report type: %v", request.ReportType))
		return response
	}
	var report, err = provider(request.Report)
	if err != nil {
		setError(response.Response, fmt.Sprint("failed to create report: %v, %v", request.ReportType, err))
		return response
	}
	manager, err := s.manager(s.config.RepositoryDatastore)
	if err != nil {
		setError(response.Response, fmt.Sprint("%v", err))
		return response
	}
	err = s.reportDao.Persist(manager, report)
	if err != nil {
		setError(response.Response, fmt.Sprint("failed to persist report %v", err))
		return response
	}
	s.reports[report.GetName()] = report
	return response
}

func (s *service) manager(datastore string) (dsc.Manager, error) {
	manager, has := s.datastores[datastore]
	if !has {
		var available = strings.Join(toolbox.MapKeysToStringSlice(s.datastores), ",")
		return nil, fmt.Errorf("failed to datastore : %v, available", datastore, available)
	}
	return manager, nil
}

func (s *service) Run(request *RunReportRequest) *RunReportResponse {
	var response = &RunReportResponse{
		Response: &Response{
			Status: "ok",
		},
	}
	report, has := s.reports[request.Name]
	if !has {
		setError(response.Response, fmt.Sprint("failed to lookup report: %v", request.Name))
		return response
	}

	manager, err := s.manager(request.Datastore)
	if err != nil {
		setError(response.Response, fmt.Sprint("%v", err))
		return response
	}

	SQL, err := report.SQL(manager, request.Parameters)
	if err != nil {
		setError(response.Response, fmt.Sprint("failed to build SQL: %v", err))
		return response
	}
	response.Data = make([]map[string]interface{}, 0)
	err = manager.ReadAll(&response.Data, SQL, nil, nil)
	if err != nil {
		setError(response.Response, fmt.Sprint("failed run query: %v %v", SQL, err))
		return response
	}
	return response
}

func (s *service) Datastores() DatastoreRegistry {
	return s.datastores
}

func (s *service) ReportProviders() ReportProviders {
	return s.providers
}

func NewService(config *Config) (Service, error) {

	if config.RepositoryDatastore == "" {
		return nil, fmt.Errorf("RepositoryDatastore was empty")
	}
	var result = &service{
		datastores: make(map[string]dsc.Manager),
		providers:  make(map[string]ReportProvider),
		reports:    make(map[string]Report),
		reportDao:  &reportDao{},
		config:     config,
	}

	for _, datastore := range config.Datastores {
		manager, err := dsc.NewManagerFactory().Create(datastore.Config)
		if err != nil {
			return nil, err
		}
		result.datastores[datastore.Name] = manager
	}
	result.providers["pivot"] = PivotReportProvider
	return result, nil
}

var defaultReportProviders = make(map[string]ReportProvider)
