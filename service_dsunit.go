package endly

import (
	"fmt"
	"github.com/viant/dsunit"
	"github.com/viant/toolbox/data"
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

//RunSQLcriptEvent represents run script event
type RunSQLcriptEvent struct {
	Datastore string
	URL       string
}

type dataStoreUnitService struct {
	*AbstractService
	Service dsunit.Service
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
  "RecreateDatastore": true,
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
			return &dsunit.RegisterRequest{}
		},
		ResponseProvider: func() interface{} {
			return &dsunit.RegisterRequest{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*dsunit.RegisterRequest); ok {
				response := s.Service.Register(handlerRequest)
				return response, response.Error()
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "recreate",
		RequestInfo: &ActionInfo{
			Description: "create datastore",
			Examples: []*ExampleUseCase{},
		},
		RequestProvider: func() interface{} {
			return &dsunit.RecreateRequest{}
		},
		ResponseProvider: func() interface{} {
			return &dsunit.RecreateResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*dsunit.RecreateRequest); ok {
				response := s.Service.Recreate(handlerRequest)
				return response, response.Error()
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "script",
		RequestInfo: &ActionInfo{
			Description: "run SQL script",
			Examples: []*ExampleUseCase{},
		},
		RequestProvider: func() interface{} {
			return &dsunit.RunScriptRequest{}
		},
		ResponseProvider: func() interface{} {
			return &dsunit.RunSQLResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*dsunit.RunScriptRequest); ok {
				response := s.Service.RunScript(handlerRequest)
				return response, response.Error()
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "sql",
		RequestInfo: &ActionInfo{
			Description: "run SQL",
			Examples: []*ExampleUseCase{},
		},
		RequestProvider: func() interface{} {
			return &dsunit.RunSQLRequest{}
		},
		ResponseProvider: func() interface{} {
			return &dsunit.RunSQLResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*dsunit.RunSQLRequest); ok {
				response := s.Service.RunSQL(handlerRequest)
				return response, response.Error()
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})


	s.Register(&ServiceActionRoute{
		Action: "mapping",
		RequestInfo: &ActionInfo{
			Description: "register database table mapping (view)",
			Examples: []*ExampleUseCase{

			},
		},
		RequestProvider: func() interface{} {
			return &dsunit.MappingRequest{}
		},
		ResponseProvider: func() interface{} {
			return &dsunit.MappingResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*dsunit.MappingRequest); ok {
				response :=  s.Service.AddTableMapping(handlerRequest)
				return response, response.Error()
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})


	s.Register(&ServiceActionRoute{
		Action: "init",
		RequestInfo: &ActionInfo{
			Description: "initialize datastore (register, recreated, run sql, add mapping)",
			Examples: []*ExampleUseCase{},
		},
		RequestProvider: func() interface{} {
			return &dsunit.InitRequest{}
		},
		ResponseProvider: func() interface{} {
			return &dsunit.InitResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*dsunit.InitRequest); ok {
				response := s.Service.Init(handlerRequest)
				return response, response.Error()
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
			return &dsunit.PrepareRequest{}
		},
		ResponseProvider: func() interface{} {
			return &dsunit.PrepareResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*dsunit.PrepareRequest); ok {
				response := s.Service.Prepare(handlerRequest)
				return response, response.Error()
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
					UseCase: "static data expect",
					Data:    dataStoreUnitServiceExpectAction,
				},
				{
					UseCase: "data expect",
					Data:    dataStoreUnitDataPrepareExaple,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &dsunit.ExpectRequest{}
		},
		ResponseProvider: func() interface{} {
			return &dsunit.ExpectResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*dsunit.ExpectRequest); ok {
				response := s.Service.Expect(handlerRequest)
				return response, response.Error()
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "query",
		RequestInfo: &ActionInfo{
			Description: "run SQL or SQL script",
		},
		RequestProvider: func() interface{} {
			return &dsunit.QueryRequest{}
		},
		ResponseProvider: func() interface{} {
			return &dsunit.QueryResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*dsunit.QueryRequest); ok {
				response := s.Service.Query(handlerRequest)
				return response, response.Error()
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
			return &dsunit.SequenceRequest{}
		},
		ResponseProvider: func() interface{} {
			return &dsunit.SequenceResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*dsunit.SequenceRequest); ok {
				response := s.Service.Sequence(handlerRequest)
				return response, response.Error()
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}


func (s dataStoreUnitService) Run(context *Context, request interface{}) *ServiceResponse {
	var state = context.state
	context.Context.Replace((*data.Map)(nil), &state)
	s.Service.SetContext(context.Context)
	return s.AbstractService.Run(context, request)
}

//context.Replace((*dsc.Manager)(nil), &manager)


//NewDataStoreUnitService creates a new Datastore unit service
func NewDataStoreUnitService() Service {
	var result = &dataStoreUnitService{
		AbstractService: NewAbstractService(DataStoreUnitServiceID),
		Service:         dsunit.New(),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
