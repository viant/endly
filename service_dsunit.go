package endly

import (
	"fmt"
	"github.com/viant/dsunit"
	"github.com/viant/dsc"
	"github.com/viant/toolbox/storage"
	"strings"
	"github.com/viant/toolbox"
	"github.com/pkg/errors"
)

const DataStoreUnitServiceId = "dsunit"


type DsUnitRegisterRequest struct {
	Datastore       string
	Config          *dsc.Config //make sure Config.Parameters have database name key
	Credential      string
	adminConfig     *dsc.Config //make sure Config.Parameters have database name key
	AdminDatastore  string      //name of admin db
	AdminCredential string
	ClearDatastore  bool
	Schema          *Resource
	DatasetMapping  *Resource
}



//DatasetMapping represnts dataset mappings
type DatasetMappings struct {
	Views map[string]*dsunit.DatasetMapping
}


func (r *DsUnitRegisterRequest) Validate() error {
	if r.Config == nil {
		return fmt.Errorf("Datastore config was nil")
	}
	if r.Config.Parameters == nil {
		r.Config.Parameters = make(map[string]string)
	}
	if r.AdminCredential == "" {
		r.AdminCredential = r.Credential
	}
	if r.AdminDatastore != "" {
		var parameters= make(map[string]string)
		toolbox.CopyMapEntries(r.Config.Parameters, parameters)
		r.adminConfig = &dsc.Config{
			DriverName: r.Config.DriverName,
			Descriptor: r.Config.Descriptor,
			Parameters: parameters,
		}
		r.adminConfig.Parameters["dbname"] = r.AdminDatastore
	}
	if _, exists := r.Config.Parameters["dbname"];! exists {
		r.Config.Parameters["dbname"] = r.Datastore
	}
	return nil
}

type DsUnitRegisterResponse struct {
	Modified int
}





type DsUnitPrepareRequest struct {
	Datasets *dsunit.DatasetResource
}



type DsUnitPrepareResponse struct {
	Added int
	Modified int
	Deleted int
}

type DsUnitVerifyRequest struct {
	Datasets *dsunit.DatasetResource
	CheckPolicy int
}

type DsUnitVerifyResponse struct {
	DatasetChecked map[string]int
}


type dsataStoreUnitService struct {
	*AbstractService
	Manager dsunit.DatasetTestManager
}

func (s *dsataStoreUnitService) Run(context *Context, request interface{}) *ServiceResponse {
	var response = &ServiceResponse{Status: "ok"}
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
	case *DsUnitVerifyRequest:
		response.Response, err = s.verify(context, actualRequest)
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

func (s *dsataStoreUnitService) registerDsManager(context *Context, datastoreName, credentialInfo string,  config *dsc.Config) error {
	credential := &storage.PasswordCredential{}
	err := LoadCredential(context.CredentialFile(credentialInfo), credential)
	if err != nil {
		return  err
	}
	config.Parameters["username"] = credential.Username
	config.Parameters["password"] = credential.Password
	config.Init()

	dsManager, err := dsc.NewManagerFactory().Create(config)
	if err != nil {
		return err
	}
	s.Manager.ManagerRegistry().Register(datastoreName, dsManager)
	return nil
}



func (s *dsataStoreUnitService) runScript(context *Context, datastore string, source *Resource) (int, error) {
	var err error
	source,err  = context.ExpandResource(source)
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
	 err := request.Validate();
	if err != nil {
		return nil, err
	}
	var result = &DsUnitRegisterResponse{}
	s.registerDsManager(context, request.Datastore, request.Credential, request.Config)
	var adminDatastore= "admin_" + request.Datastore
	if request.adminConfig != nil {
		s.registerDsManager(context, adminDatastore, request.AdminCredential, request.adminConfig)
	}
	if request.ClearDatastore {
		err := s.Manager.ClearDatastore(adminDatastore, request.Datastore)
		if err != nil {
			return nil, err
		}
	}
	if request.Schema != nil {
		result.Modified, err = s.runScript(context, request.Datastore, request.Schema)
		if err != nil {
			return nil, err
		}
	}

	if request.DatasetMapping != nil {
		mappingResource, err := context.ExpandResource(request.DatasetMapping)
		if err != nil {
			return nil, err
		}
		var datasetMapping =&DatasetMappings{}
		err = mappingResource.JsonDecode(datasetMapping)
		if err != nil {
			return nil, err
		}
		for view, mapping := range datasetMapping.Views {
			s.Manager.RegisterDatasetMapping(view, mapping)
		}
	}
	return result, nil
}


func (s *dsataStoreUnitService) prepare(context *Context, request *DsUnitPrepareRequest) (interface{}, error) {
	var response = &DsUnitPrepareResponse{}
	datasets, err := s.Manager.DatasetFactory().CreateDatasets(request.Datasets)
	if err != nil {
		return nil, err
	}
	response.Added, response.Modified, response.Deleted, err = s.Manager.PrepareDatastore(datasets)
	return response, err
}

func (s *dsataStoreUnitService) verify(context *Context, request *DsUnitVerifyRequest) (interface{}, error) {
	var response = &DsUnitVerifyResponse{
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
	case "prepare":
		return &DsUnitPrepareRequest{}, nil
	case "verify":
		return &DsUnitVerifyRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

func NewDataStoreUnitService() Service {
	var result = &dsataStoreUnitService{
		AbstractService: NewAbstractService(DataStoreUnitServiceId),
		Manager:         dsunit.NewDatasetTestManager(),
	}
	result.Manager.SafeMode(false)
	result.AbstractService.Service = result
	return result
}
