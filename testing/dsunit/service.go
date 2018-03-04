package dsunit

import (
	"fmt"
	"github.com/viant/dsunit"
	"github.com/viant/toolbox/data"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
)

var converter = toolbox.NewColumnConverter("yyyy-MM-dd HH:ss")

const (
	//ServiceID represents a data store unit service id
	ServiceID = "dsunit"
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

type service struct {
	*endly.AbstractService
	Service dsunit.Service
}

const (
	dsunitMySQLInitExample = `{
  "Datastore": "mydb",
  "Config": {
    "DriverName": "mysql",
    "Descriptor": "[username]:[password]@tcp(127.0.0.1:3306)/[dbname]?parseTime=true",
    "Credential": "$mysqlCredential"
  },
  "Admin": {
    "Datastore": "mysql",
    "Config": {
      "DriverName": "mysql",
      "Descriptor": "[username]:[password]@tcp(127.0.0.1:3306)/[dbname]?parseTime=true",
      "Credential": "$mysqlCredential"
    }
  },
  "Scripts": [
    {
      "URL": "datastore/mydb/schema.ddl"
    }
  ],
  "Recreate": "true"
}`

	dsunitAerospikeRegisterExample = `{
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
  }
}`

	dsunitBigQueryRegisterExample = `{
  "Datastore": "db1",
  "Config": {
    "DriverName": "bigquery",
    "Descriptor": "bq/[datasetId]",
	"Credential": "${env.HOME}/.secret/bq.json",
    "Parameters": {
      "datasetId": "db1",
      "dateFormat": "yyyy-MM-dd HH:mm:ss.SSSZ",
      "projectId": "xxxxx"
    }
  }
	
}`

	dsunitMySQLRegisterExample = `{
  "Datastore": "db1",
  "Config": {
    "DriverName": "mysql",
  	"Credential": "${env.HOME}/.secret/mysql.json",
    "Descriptor": "[username]:[password]@tcp(127.0.0.1:3306)/[dbname]?parseTime=true"
  }
}`

	dsunitServiceSQLExample = `{
		"Datastore": "db1",
		"Scripts": [
			{
				"URL": "datastore/db1/schema.ddl"
			}
		]
	}`

	dsunitServiceMappingExample = ` {
		"Mappings": [
			{
				"URL": "config/mapping/v_asset.json"
			}
		]
	}`

	dsunitServiceSequenceExample = `{
		"Datastore": "db1",
		"Tables": [
			"table1",
			"table2"
		]
	}`

	dsunitServiceStaticDataPrepareExample = `{
    "Datastore": "db1",
    "URL": "datastore/db1/dictionary"
  }`

	dsunitDataPrepareExaple = ` {
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

	dsunitServiceExpectAction = `{
    "Datastore": "db1",
    "URL": "datastore/db1/use_case2/",
	"Prefix":"expect_"
  }`
)

func (s *service) registerRoutes() {

	s.Register(&endly.ServiceActionRoute{
		Action: "register",
		RequestInfo: &endly.ActionInfo{
			Description: "register database connection",
			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "aerospike datastore registration",
					Data:    dsunitAerospikeRegisterExample,
				},
				{
					UseCase: "BigQuery datastore registration",
					Data:    dsunitBigQueryRegisterExample,
				},

				{
					UseCase: "MySQL datastore registration",
					Data:    dsunitMySQLRegisterExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &dsunit.RegisterRequest{}
		},
		ResponseProvider: func() interface{} {
			return &dsunit.RegisterRequest{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*dsunit.RegisterRequest); ok {
				response := s.Service.Register(req)
				return response, response.Error()
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "recreate",
		RequestInfo: &endly.ActionInfo{
			Description: "create datastore",
			Examples:    []*endly.ExampleUseCase{},
		},
		RequestProvider: func() interface{} {
			return &dsunit.RecreateRequest{}
		},
		ResponseProvider: func() interface{} {
			return &dsunit.RecreateResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*dsunit.RecreateRequest); ok {
				response := s.Service.Recreate(req)
				return response, response.Error()
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "script",
		RequestInfo: &endly.ActionInfo{
			Description: "run SQL script",
			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "run script",
					Data:    dsunitServiceSQLExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &dsunit.RunScriptRequest{}
		},
		ResponseProvider: func() interface{} {
			return &dsunit.RunSQLResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*dsunit.RunScriptRequest); ok {
				response := s.Service.RunScript(req)
				return response, response.Error()
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "sql",
		RequestInfo: &endly.ActionInfo{
			Description: "run SQL",
			Examples:    []*endly.ExampleUseCase{},
		},
		RequestProvider: func() interface{} {
			return &dsunit.RunSQLRequest{}
		},
		ResponseProvider: func() interface{} {
			return &dsunit.RunSQLResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*dsunit.RunSQLRequest); ok {
				response := s.Service.RunSQL(req)
				return response, response.Error()
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "mapping",
		RequestInfo: &endly.ActionInfo{
			Description: "register database table mapping (view)",
			Examples:    []*endly.ExampleUseCase{},
		},
		RequestProvider: func() interface{} {
			return &dsunit.MappingRequest{}
		},
		ResponseProvider: func() interface{} {
			return &dsunit.MappingResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*dsunit.MappingRequest); ok {
				response := s.Service.AddTableMapping(req)
				return response, response.Error()
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "init",
		RequestInfo: &endly.ActionInfo{
			Description: "initialize datastore (register, recreated, run sql, add mapping)",

			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "mysql init",
					Data:    dsunitMySQLInitExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &dsunit.InitRequest{}
		},
		ResponseProvider: func() interface{} {
			return &dsunit.InitResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*dsunit.InitRequest); ok {
				response := s.Service.Init(req)
				return response, response.Error()
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "prepare",
		RequestInfo: &endly.ActionInfo{
			Description: "populate databstore with provided data",
			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "static data prepare",
					Data:    dsunitServiceStaticDataPrepareExample,
				},
				{
					UseCase: "data prepare",
					Data:    dsunitDataPrepareExaple,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &dsunit.PrepareRequest{
				DatasetResource: &dsunit.DatasetResource{
					DatastoreDatasets: &dsunit.DatastoreDatasets{},
				},
			}
		},
		ResponseProvider: func() interface{} {
			return &dsunit.PrepareResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*dsunit.PrepareRequest); ok {
				response := s.Service.Prepare(req)
				return response, response.Error()
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "expect",
		RequestInfo: &endly.ActionInfo{
			Description: "verify databstore with provided data",
			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "static data expect",
					Data:    dsunitServiceExpectAction,
				},
				{
					UseCase: "data expect",
					Data:    dsunitDataPrepareExaple,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &dsunit.ExpectRequest{
				DatasetResource: &dsunit.DatasetResource{
					DatastoreDatasets: &dsunit.DatastoreDatasets{},
				},
			}
		},
		ResponseProvider: func() interface{} {
			return &dsunit.ExpectResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*dsunit.ExpectRequest); ok {
				response := s.Service.Expect(req)
				return response, response.Error()
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "query",
		RequestInfo: &endly.ActionInfo{
			Description: "run SQL query",
		},
		RequestProvider: func() interface{} {
			return &dsunit.QueryRequest{}
		},
		ResponseProvider: func() interface{} {
			return &dsunit.QueryResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*dsunit.QueryRequest); ok {
				response := s.Service.Query(req)
				return response, response.Error()
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.ServiceActionRoute{
		Action: "sequence",
		RequestInfo: &endly.ActionInfo{
			Description: "get sequence for supplied tables",
			Examples: []*endly.ExampleUseCase{
				{
					UseCase: "sequence",
					Data:    dsunitServiceSequenceExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &dsunit.SequenceRequest{}
		},
		ResponseProvider: func() interface{} {
			return &dsunit.SequenceResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*dsunit.SequenceRequest); ok {
				response := s.Service.Sequence(req)
				return response, response.Error()
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

func (s service) Run(context *endly.Context, request interface{}) *endly.ServiceResponse {
	var state = context.State()
	context.Context.Replace((*data.Map)(nil), &state)
	s.Service.SetContext(context.Context)
	return s.AbstractService.Run(context, request)
}


//New creates a new Datastore unit service
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
		Service:         dsunit.New(),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
