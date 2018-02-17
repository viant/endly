package endly

import (
	"fmt"
	"github.com/viant/assertly"
	"github.com/viant/dsc"
	"github.com/viant/dsunit"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
	"strings"
)

const (
	//DataStoreUnitServiceID represents a data store unit service id
	DataStoreUnitServiceID = "dsunit"
)

//PopulateDatastoreEvent represents a populate Datastore event
type PopulateDatastoreEvent struct {
	Datastore string `required:"true" description:"register datastore name"` //target host
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

func (s *dataStoreUnitService) getSequences(context *Context, request *DsUnitTableSequenceRequest) (*DsUnitTableSequenceResponse, error) {
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

func (s *dataStoreUnitService) registerDsManager(context *Context, datastoreName, credential string, config *dsc.Config) (dsc.Manager, error) {
	credentialConfig := &cred.Config{}

	if credential != "" {
		err := credentialConfig.Load(credential)
		if err != nil {
			return nil, err
		}
	}
	config.Parameters["username"] = credentialConfig.Username
	config.Parameters["password"] = credentialConfig.Password
	err := config.Init()
	if err != nil {
		return nil, err
	}
	dsManager, err := dsc.NewManagerFactory().Create(config)
	if err != nil {
		return nil, err
	}
	s.Manager.ManagerRegistry().Register(datastoreName, dsManager)
	return dsManager, nil
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

func (s *dataStoreUnitService) runScripts(context *Context, request *DsUnitSQLRequest) (*DsUnitSQLResponse, error) {
	var err error
	var response = &DsUnitSQLResponse{}
	response.Modified, err = s.runSQLScripts(context, request.Datastore, request.Scripts)
	if err != nil {
		return nil, err
	}

	if len(request.SQLs) > 0 {
		for _, SQL := range request.SQLs {
			modified, err := s.runSQL(context, request.Datastore, SQL)
			if err != nil {
				return nil, err
			}
			response.Modified += modified
		}
	}
	return response, nil
}

func (s *dataStoreUnitService) runSQL(context *Context, datastore string, SQLs string) (int, error) {
	scriptRequest := &dsunit.Script{
		Datastore: datastore,
		Sqls:      dsunit.ParseSQLScript(strings.NewReader(SQLs)),
	}
	return s.Manager.Execute(scriptRequest)
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
	return s.runSQL(context, datastore, script)
}

func (s *dataStoreUnitService) registerTables(state data.Map, dsManger dsc.Manager, tables []*dsc.TableDescriptor) {
	if len(tables) == 0 {
		return
	}
	for _, table := range tables {
		table.Table = state.ExpandAsText(table.Table)
		dsManger.TableDescriptorRegistry().Register(table)
	}

}

func (s *dataStoreUnitService) register(context *Context, request *DsUnitRegisterRequest) (interface{}, error) {
	request.Init()
	var state = context.state
	var dsManager dsc.Manager
	var result = &DsUnitRegisterResponse{}
	dsManager, err := s.registerDsManager(context, request.Datastore, request.Credential, request.Config)
	if err != nil {
		return nil, err
	}

	s.registerTables(state, dsManager, request.Tables)

	var adminDatastore = "admin_" + request.Datastore
	if request.adminConfig != nil {
		dsManager, err = s.registerDsManager(context, adminDatastore, request.AdminCredential, request.adminConfig)
		if err != nil {
			return nil, err
		}
		s.registerTables(state, dsManager, request.Tables)
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
			data.Table = state.ExpandAsText(data.Table)
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
	datasets, err := s.buildDatasets(context, request.AsDatasetResource(), request.Expand)
	if err != nil {
		return nil, err
	}
	for _, dataSet := range datasets.Datasets {
		var populateDatastoreEvent = &PopulateDatastoreEvent{Datastore: request.Datastore, Table: dataSet.Table, Rows: len(dataSet.Rows)}
		AddEvent(context, populateDatastoreEvent, Pairs("value", populateDatastoreEvent), Info)
	}

	response.Added, response.Modified, response.Deleted, err = s.Manager.PrepareDatastore(datasets)
	return response, err
}

func (s *dataStoreUnitService) verify(context *Context, request *DsUnitExpectRequest) (response *DsUnitExpectResponse, err error) {
	datasets, err := s.buildDatasets(context, request.AsDatasetResource(), request.Expand)
	if err != nil {
		return nil, err
	}
	response = &DsUnitExpectResponse{
		Validation: &assertly.Validation{},
	}

	var verificationFailures = make(map[string]bool)
	violations, err := s.Manager.ExpectDatasets(request.CheckPolicy, datasets)
	if err != nil {
		return nil, err
	}

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
			response.AddFailure(assertly.NewFailure(path, message, violation.Expected, violation.Actual))
		}
	}
	for _, dataset := range datasets.Datasets {
		if verificationFailures[dataset.Table] {
			continue
		}
		response.PassedCount = len(dataset.Rows)
	}
	return response, err
}

const (
	dataStoreUnitAerospikeRegisterExample = `{
  "Datastore": "db",
  "Config": {
    "DriverName": "aerospike",
    "Descriptor": "tcp([host]:3000)/[namespace]",
    "Parameters": {
      "dateFormat": "yyyy-MM-dd hh:mm:ss",
      "dbname": "db",
      "host": "127.0.0.1",
      "keyColumnName": "id",
      "namespace": "db",
      "port": "3000"
    }
  },
  "AdminDatastore": "db"
}`

	dataStoreUnitBigQueryRegisterExample = `{
  "Datastore": "db1",
  "Config": {
    "DriverName": "bigquery",
    "Descriptor": "bq/[datasetId]",
    "Parameters": {
      "credentialsFile": "${env.HOME}/.secret/bq.json",
      "datasetId": "db1",
      "dateFormat": "yyyy-MM-dd HH:mm:ss.SSSZ",
      "projectId": "xxxxx"
    }
  },
  "AdminDatastore": "db1",
  "ClearDatastore": true,
  "Tables": [
    {
      "Table": "my_table",
      "PkColumns": [
        "id"
      ],
      "SchemaUrl": ""
    }
  ]
}`

	dataStoreUnitMySQLRegisterExample = `{
  "Datastore": "db1",
  "Config": {
    "DriverName": "mysql",
    "Descriptor": "[username]:[password]@tcp(127.0.0.1:3306)/[dbname]?parseTime=true"
  },
  "Credential": "${env.HOME}/.secret/mysql.json"
}`
	dataStoreUnitServiceSQLExample = `{
		"Datastore": "db1",
		"Scripts": [
			{
				"URL": "datastore/db1/schema.ddl"
			}
		]
	}`

	dataStoreUnitServiceMappingExample = ` {
		"Mappings": [
			{
				"URL": "config/mapping/v_asset.json"
			}
		]
	}`

	dataStoreUnitServiceSequenceExample = `{
		"Datastore": "db1",
		"Tables": [
			"table1",
			"table2"
		]
	}`

	dataStoreUnitServiceStaticDataPrepareExample = `{
    "Datastore": "db1",
    "URL": "datastore/db1/dictionary"
  }`

	dataStoreUnitDataPrepareExaple = ` {
		"Datastore": "db1",
		"Data": {
			"table1": [
				{
					"id": 1,
					"name": "test 1",
					"type": "pivot"
				},
				{
					"id": 2,
					"name": "test 2",
					"type": "pivot"
				}
			],
			"table2": [
				{
					"id": 1,
					"name": "test 1",
					"type": "pivot"
				},
				{
					"id": 2,
					"name": "test 2",
					"type": "pivot"
				}
			]
		}
	}`

	dataStoreUnitServiceExpectAction = `{
    "Datastore": "db1",
    "URL": "datastore/db1/use_case2/",
	"Prefix":"expect_"
  }`
)

func (s *dataStoreUnitService) registerRoutes() {

	s.Register(&ServiceActionRoute{
		Action: "register",
		RequestInfo: &ActionInfo{
			Description: "register database connection",
			Examples: []*ExampleUseCase{
				{
					UseCase: "aerospike datastore registration",
					Data:    dataStoreUnitAerospikeRegisterExample,
				},
				{
					UseCase: "BigQuery datastore registration",
					Data:    dataStoreUnitBigQueryRegisterExample,
				},

				{
					UseCase: "MySQL datastore registration",
					Data:    dataStoreUnitMySQLRegisterExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &DsUnitRegisterRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DsUnitRegisterResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*DsUnitRegisterRequest); ok {
				return s.register(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "sql",
		RequestInfo: &ActionInfo{
			Description: "run SQL or SQL script",
			Examples: []*ExampleUseCase{
				{
					UseCase: "sql",
					Data:    dataStoreUnitServiceSQLExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &DsUnitSQLRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DsUnitSQLResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*DsUnitSQLRequest); ok {
				return s.runScripts(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "mapping",
		RequestInfo: &ActionInfo{
			Description: "register database table mapping (view)",
			Examples: []*ExampleUseCase{
				{
					UseCase: "sql",
					Data:    dataStoreUnitServiceMappingExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &DsUnitMappingRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DsUnitMappingResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*DsUnitMappingRequest); ok {
				return s.addMapping(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "sequence",
		RequestInfo: &ActionInfo{
			Description: "get sequence for supplied tables",
			Examples: []*ExampleUseCase{
				{
					UseCase: "sequence",
					Data:    dataStoreUnitServiceSequenceExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &DsUnitTableSequenceRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DsUnitTableSequenceResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*DsUnitTableSequenceRequest); ok {
				return s.getSequences(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "prepare",
		RequestInfo: &ActionInfo{
			Description: "populate databstore with provided data",
			Examples: []*ExampleUseCase{
				{
					UseCase: "static data prepare",
					Data:    dataStoreUnitServiceStaticDataPrepareExample,
				},
				{
					UseCase: "data prepare",
					Data:    dataStoreUnitDataPrepareExaple,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &DsUnitPrepareRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DsUnitPrepareResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*DsUnitPrepareRequest); ok {
				return s.prepare(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "expect",
		RequestInfo: &ActionInfo{
			Description: "verify databstore with provided data",
			Examples: []*ExampleUseCase{
				{
					UseCase: "static data exppect",
					Data:    dataStoreUnitServiceExpectAction,
				},
				{
					UseCase: "data expect",
					Data:    dataStoreUnitDataPrepareExaple,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &DsUnitExpectRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DsUnitExpectResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*DsUnitExpectRequest); ok {
				return s.verify(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

}

//NewDataStoreUnitService creates a new Datastore unit service
func NewDataStoreUnitService() Service {
	var result = &dataStoreUnitService{
		AbstractService: NewAbstractService(DataStoreUnitServiceID),
		Manager:         dsunit.NewDatasetTestManager(),
	}
	result.Manager.SafeMode(false)
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
