package endly

import (
	"fmt"
	"github.com/viant/dsc"
	"github.com/viant/dsunit"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/url"
	"strings"
)

const (
	//DataStoreUnitServiceID represents a data store unit service id
	DataStoreUnitServiceID = "dsunit"

	//DataStoreUnitServiceRegisterAction represents datastore dsn register action
	DataStoreUnitServiceRegisterAction = "register"

	//DataStoreUnitServiceSQLAction represents sql run action
	DataStoreUnitServiceSQLAction = "sql"

	//DataStoreUnitServiceMappingAction represents tables mapping action
	DataStoreUnitServiceMappingAction = "mapping"

	//DataStoreUnitServicePrepareAction represents datastore data preparation action
	DataStoreUnitServicePrepareAction = "prepare"

	//DataStoreUnitServiceSequenceAction represents datastore reading sequence action
	DataStoreUnitServiceSequenceAction = "sequence"

	//DataStoreUnitServiceExpectAction represents datastore data verification action
	DataStoreUnitServiceExpectAction = "expect"
)

//PopulateDatastoreEvent represents a populate Datastore event
type PopulateDatastoreEvent struct {
	Datastore string
	Table     string
	Rows      int
}

//RunSQLScriptEvent represents run script event
type RunSQLScriptEvent struct {
	Datastore string
	URL       string
}

type dataStoreUnitService struct {
	*AbstractService
	Manager dsunit.DatasetTestManager
}

func (s *dataStoreUnitService) Run(context *Context, request interface{}) *ServiceResponse {
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

	case *DsUnitSQLScriptRequest:
		response.Response, err = s.runScripts(context, actualRequest)
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
		response.Error = fmt.Sprintf("unsupported request type: %T", request)
	}
	if response.Error != "" {
		response.Status = "err"
	}
	return response
}

func (s *dataStoreUnitService) getSequences(context *Context, request *DsUnitTableSequenceRequest) (*DsUnitTableSequenceResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	manager := s.Manager.ManagerRegistry().Get(request.Datastore)
	if manager == nil {
		return nil, fmt.Errorf("unknown Datastore: %v", request.Datastore)
	}
	var response = &DsUnitTableSequenceResponse{
		Sequences: make(map[string]int),
	}
	dbConfig := manager.Config()
	var sequence int64
	var err error
	dialect := dsc.GetDatastoreDialect(dbConfig.DriverName)
	for _, table := range request.Tables {
		sequence, err = dialect.GetSequence(manager, table)
		response.Sequences[table] = int(sequence)
	}
	if len(response.Sequences) == 0 {
		return response, err
	}
	return response, nil
}

func (s *dataStoreUnitService) registerDsManager(context *Context, datastoreName, credential string, config *dsc.Config) error {
	credentialConfig := &cred.Config{}

	if credential != "" {
		err := credentialConfig.Load(credential)
		if err != nil {
			return err
		}
	}
	config.Parameters["username"] = credentialConfig.Username
	config.Parameters["password"] = credentialConfig.Password
	err := config.Init()
	if err != nil {
		return err
	}
	dsManager, err := dsc.NewManagerFactory().Create(config)
	if err != nil {
		return err
	}
	s.Manager.ManagerRegistry().Register(datastoreName, dsManager)
	return nil
}

func (s *dataStoreUnitService) addMapping(context *Context, request *DsUnitMappingRequest) (*DsUnitMappingResponse, error) {
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
				return nil, fmt.Errorf("failed to decode: %v %v", mappingResource.URL, err)
			}
			response.Tables = append(response.Tables, datasetMapping.Value.Tables()...)
			s.Manager.RegisterDatasetMapping(datasetMapping.Name, datasetMapping.Value)
		}
	}
	return response, nil
}

func (s *dataStoreUnitService) runScripts(context *Context, request *DsUnitSQLScriptRequest) (*DsUnitSQLScriptResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	var err error
	var response = &DsUnitSQLScriptResponse{}
	response.Modified, err = s.runSQLScripts(context, request.Datastore, request.Scripts)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (s *dataStoreUnitService) runSQLScripts(context *Context, datastore string, scripts []*url.Resource) (int, error) {
	if len(scripts) == 0 {
		return 0, nil
	}
	var totaModified = 0
	for _, script := range scripts {
		var event = &RunSQLScriptEvent{Datastore: datastore, URL: script.URL}
		AddEvent(context, event, Pairs("value", event), Info)
		modified, err := s.loadSQLAndRun(context, datastore, script)
		if err != nil {
			return 0, err
		}
		totaModified += modified
	}
	return totaModified, nil
}

func (s *dataStoreUnitService) loadSQLAndRun(context *Context, datastore string, source *url.Resource) (int, error) {
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

func (s *dataStoreUnitService) register(context *Context, request *DsUnitRegisterRequest) (interface{}, error) {
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

	result.Modified, err = s.runSQLScripts(context, request.Datastore, request.Scripts)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *dataStoreUnitService) buildDatasets(context *Context, datasetResource *dsunit.DatasetResource, expand bool) (*dsunit.Datasets, error) {
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

	if expand {
		var state = context.State()
		for _, data := range datasets.Datasets {
			for _, row := range data.Rows {
				expanded := state.Expand(row.Values)
				row.Values = toolbox.AsMap(expanded)
			}
		}
	}
	return datasets, err
}

func (s *dataStoreUnitService) prepare(context *Context, request *DsUnitPrepareRequest) (interface{}, error) {
	var response = &DsUnitPrepareResponse{}
	err := request.Validate()
	if err != nil {
		return nil, err
	}
	datasets, err := s.buildDatasets(context, request.AsDatasetResource(), request.Expand)
	if err != nil {
		return nil, err
	}
	for _, data := range datasets.Datasets {
		var populateDatastoreEvent = &PopulateDatastoreEvent{Datastore: request.Datastore, Table: data.Table, Rows: len(data.Rows)}
		AddEvent(context, populateDatastoreEvent, Pairs("value", populateDatastoreEvent), Info)
	}

	response.Added, response.Modified, response.Deleted, err = s.Manager.PrepareDatastore(datasets)
	return response, err
}

func (s *dataStoreUnitService) verify(context *Context, request *DsUnitExpectRequest) (*ValidationInfo, error) {
	err := request.Validate()
	if err != nil {
		return nil, err
	}
	datasets, err := s.buildDatasets(context, request.AsDatasetResource(), request.Expand)
	if err != nil {
		return nil, err
	}
	var verificationFailures = make(map[string]bool)
	violations, err := s.Manager.ExpectDatasets(request.CheckPolicy, datasets)
	if err != nil {
		return nil, err
	}
	validationInfo := &ValidationInfo{}

	if violations.HasViolations() {
		for _, violation := range violations.Violations() {
			verificationFailures[violation.Table] = true
			var path = fmt.Sprintf("%v%v", violation.Table, violation.Key)
			var message = ""

			switch violation.Type {
			case dsunit.ViolationTypeInvalidRowCount:
				message += fmt.Sprintf("expected %v rows but had %v\n\t", violation.Expected, violation.Actual)
			case dsunit.ViolationTypeMissingActualRow:
				message += "The following row was missing:\n\t\t"
				message += fmt.Sprintf("[PK: %v]: %v \n\t\t", violation.Key, violation.Expected)
			case dsunit.ViolationTypeRowNotEqual:
				message += "\n\tThe following row was different:\n\t\t"
				message += fmt.Sprintf("[PK: %v]:  %v !=  actual: %v \n\t\t", violation.Key, violation.Expected, violation.Actual)
			}
			validationInfo.AddFailure(NewFailedTest(path, message, violation.Expected, violation.Actual))
		}

	}

	for _, dataset := range datasets.Datasets {
		if verificationFailures[dataset.Table] {
			continue
		}
		validationInfo.TestPassed = len(dataset.Rows)
	}
	return validationInfo, err
}

func (s *dataStoreUnitService) NewRequest(action string) (interface{}, error) {
	switch action {
	case DataStoreUnitServiceRegisterAction:
		return &DsUnitRegisterRequest{}, nil
	case DataStoreUnitServiceSQLAction:
		return &DsUnitSQLScriptRequest{}, nil
	case DataStoreUnitServiceMappingAction:
		return &DsUnitMappingRequest{}, nil
	case DataStoreUnitServicePrepareAction:
		return &DsUnitPrepareRequest{}, nil
	case DataStoreUnitServiceSequenceAction:
		return &DsUnitTableSequenceRequest{}, nil
	case DataStoreUnitServiceExpectAction:
		return &DsUnitExpectRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

//NewDataStoreUnitService creates a new Datastore unit service
func NewDataStoreUnitService() Service {
	var result = &dataStoreUnitService{
		AbstractService: NewAbstractService(DataStoreUnitServiceID,
			DataStoreUnitServiceRegisterAction,
			DataStoreUnitServiceSQLAction,
			DataStoreUnitServiceMappingAction,
			DataStoreUnitServicePrepareAction,
			DataStoreUnitServiceSequenceAction,
			DataStoreUnitServiceExpectAction,
		),
		Manager: dsunit.NewDatasetTestManager(),
	}
	result.Manager.SafeMode(false)
	result.AbstractService.Service = result
	return result
}
