package endly

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/viant/dsc"
	"github.com/viant/dsunit"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/url"
	"strings"
)

//DataStoreUnitServiceID represents a data store unit service id
const DataStoreUnitServiceID = "dsunit"

type dsataStoreUnitService struct {
	*AbstractService
	Manager dsunit.DatasetTestManager
}

func (s *dsataStoreUnitService) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok"}
	defer s.End(context)(startEvent, Pairs("response", response))
	var err error
	switch actualRequest := request.(type) {
	case *DsUnitRegisterRequest:
		response.Response, err = s.register(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("%v", err)
		}

	case *DsUnitPrepareRequest:
		response.Response, err = s.prepare(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("%v", err)
		}
	case *DsUnitExpectRequest:
		response.Response, err = s.verify(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("%v", err)
		}

	case *DsUnitTableSequenceRequest:
		response.Response, err = s.getSequences(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("%v", err)
		}

	case *DsUnitMappingRequest:
		response.Response, err = s.addMapping(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("%v", err)
		}

	default:
		response.Error = fmt.Sprintf("Unsupported request type: %T", request)
	}
	if response.Error != "" {
		response.Status = "err"
	}
	return response
}

func (s *dsataStoreUnitService) getSequences(context *Context, request *DsUnitTableSequenceRequest) (*DsUnitTableSequenceResponse, error) {
	manager := s.Manager.ManagerRegistry().Get(request.Datastore)
	if manager == nil {
		return nil, fmt.Errorf("Unknown datastore: %v", request.Datastore)
	}
	var response = &DsUnitTableSequenceResponse{
		Sequences: make(map[string]int),
	}
	dbConfig := manager.Config()
	dialect := dsc.GetDatastoreDialect(dbConfig.DriverName)
	for _, table := range request.Tables {
		sequence, _ := dialect.GetSequence(manager, table)
		response.Sequences[table] = int(sequence)
	}
	return response, nil
}

func (s *dsataStoreUnitService) registerDsManager(context *Context, datastoreName, credential string, config *dsc.Config) error {
	credentialConfig := &cred.Config{}

	if credential != "" {
		err := credentialConfig.Load(credential)
		if err != nil {
			return err
		}
	}
	config.Parameters["username"] = credentialConfig.Username
	config.Parameters["password"] = credentialConfig.Password
	config.Init()

	dsManager, err := dsc.NewManagerFactory().Create(config)
	if err != nil {
		return err
	}
	s.Manager.ManagerRegistry().Register(datastoreName, dsManager)
	return nil
}

func (s *dsataStoreUnitService) addMapping(context *Context, request *DsUnitMappingRequest) (*DsUnitMappingResponse, error) {
	var response = &DsUnitMappingResponse{
		Tables: make([]string, 0),
	}
	if request.Mappings != nil {
		for _, mapping := range request.Mappings {
			mappingResource, err := context.ExpandResource(mapping)
			if err != nil {
				return nil, err
			}
			var datasetMapping = &DatasetMapping{}
			err = mappingResource.JSONDecode(datasetMapping)
			if err != nil {
				return nil, fmt.Errorf("Failed to decode: %v %v", mappingResource.URL, err)
			}
			response.Tables = append(response.Tables, datasetMapping.Value.Tables()...)
			s.Manager.RegisterDatasetMapping(datasetMapping.Name, datasetMapping.Value)
		}
	}
	return response, nil
}

func (s *dsataStoreUnitService) runScript(context *Context, datastore string, source *url.Resource) (int, error) {
	var err error

	source, err = context.ExpandResource(source)
	if err != nil {
		return 0, err
	}
	script, err := source.DownloadText()
	if err != nil {
		return 0, err
	}
	scriptRequest := &dsunit.Script{
		Datastore: datastore,
		Sqls:      dsunit.ParseSQLScript(strings.NewReader(script)),
	}
	return s.Manager.Execute(scriptRequest)
}

func (s *dsataStoreUnitService) register(context *Context, request *DsUnitRegisterRequest) (interface{}, error) {
	err := request.Validate()
	if err != nil {
		return nil, err
	}
	var result = &DsUnitRegisterResponse{}
	err = s.registerDsManager(context, request.Datastore, request.Credential, request.Config)
	if err != nil {
		return nil, err
	}
	var adminDatastore = "admin_" + request.Datastore
	if request.adminConfig != nil {
		err = s.registerDsManager(context, adminDatastore, request.AdminCredential, request.adminConfig)
		if err != nil {
			return nil, err
		}
	}
	if request.ClearDatastore {
		err := s.Manager.ClearDatastore(adminDatastore, request.Datastore)
		if err != nil {
			return nil, err
		}
	}

	if len(request.Scripts) > 0 {
		for _, script := range request.Scripts {
			s.AddEvent(context, "SQLScript", Pairs("datastore", request.Datastore, "url", script.URL), Info)
			result.Modified, err = s.runScript(context, request.Datastore, script)

			if err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

func (s *dsataStoreUnitService) prepare(context *Context, request *DsUnitPrepareRequest) (interface{}, error) {
	var response = &DsUnitPrepareResponse{}
	err := request.Validate()
	if err != nil {
		return nil, err
	}

	datasetResource := request.AsDatasetResource()

	if datasetResource.URL != "" {
		resource, err := context.ExpandResource(&url.Resource{URL: datasetResource.URL})
		if err != nil {
			return nil, err
		}
		datasetResource.URL = resource.URL
	}
	datasets, err := s.Manager.DatasetFactory().CreateDatasets(datasetResource)
	if err != nil {
		return nil, err
	}

	if request.Expand {
		var state = context.State()
		for _, data := range datasets.Datasets {
			for _, row := range data.Rows {
				expanded := state.Expand(row.Values)
				row.Values = toolbox.AsMap(expanded)
			}
		}
	}

	for _, data := range datasets.Datasets {
		s.AddEvent(context, "PopulateDatastore", Pairs("datastore", request.Datastore, "table", data.Table, "rows", len(data.Rows)), Info)
	}

	response.Added, response.Modified, response.Deleted, err = s.Manager.PrepareDatastore(datasets)
	return response, err
}

func (s *dsataStoreUnitService) verify(context *Context, request *DsUnitExpectRequest) (interface{}, error) {
	err := request.Validate()
	if err != nil {
		return nil, err
	}
	var response = &DsUnitExpectResponse{
		DatasetChecked: make(map[string]int),
	}

	datasets, err := s.Manager.DatasetFactory().CreateDatasets(request.Datasets)
	if err != nil {
		return nil, err
	}
	for _, dataset := range datasets.Datasets {
		response.DatasetChecked[dataset.Table] = len(dataset.Rows)
	}

	voilations, err := s.Manager.ExpectDatasets(request.CheckPolicy, datasets)
	if err != nil {
		return nil, err
	}
	if voilations.HasViolations() {
		return nil, errors.New(voilations.String())
	}
	return response, err
}

func (s *dsataStoreUnitService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "register":
		return &DsUnitRegisterRequest{}, nil
	case "mapping":
		return &DsUnitMappingRequest{}, nil
	case "prepare":
		return &DsUnitPrepareRequest{}, nil
	case "sequence":
		return &DsUnitTableSequenceRequest{}, nil
	case "expect":
		return &DsUnitExpectRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

//NewDataStoreUnitService creates a new datastore unit service
func NewDataStoreUnitService() Service {
	var result = &dsataStoreUnitService{
		AbstractService: NewAbstractService(DataStoreUnitServiceID),
		Manager:         dsunit.NewDatasetTestManager(),
	}
	result.Manager.SafeMode(false)
	result.AbstractService.Service = result
	return result
}
