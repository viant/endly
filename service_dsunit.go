package endly

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/viant/dsc"
	"github.com/viant/dsunit"
	"github.com/viant/endly/common"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/storage"
	"strings"
)

const DataStoreUnitServiceId = "dsunit"

type DsUnitRegisterRequest struct {
	Datastore       string
	Config          *dsc.Config //make sure Deploy.Parameters have database name key
	Credential      string
	adminConfig     *dsc.Config //make sure Deploy.Parameters have database name key
	AdminDatastore  string      //name of admin db
	AdminCredential string
	ClearDatastore  bool
	Scripts         []*Resource
}

//DatasetMapping represnts dataset mappings
type DatasetMappings struct {
	Name  string                 //mapping name
	Value *dsunit.DatasetMapping //actual mappings
}

type DsUnitMappingRequest struct {
	Mappings []*Resource
}

type DsUnitMappingResonse struct {
	Tables []string
}

type DsUnitTableSequenceRequest struct {
	Datastore string
	Tables    []string
}

type DsUnitTableSequenceResponse struct {
	Sequences map[string]int
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
		var parameters = make(map[string]string)
		toolbox.CopyMapEntries(r.Config.Parameters, parameters)
		r.adminConfig = &dsc.Config{
			DriverName: r.Config.DriverName,
			Descriptor: r.Config.Descriptor,
			Parameters: parameters,
		}
		r.adminConfig.Parameters["dbname"] = r.AdminDatastore
	}
	if _, exists := r.Config.Parameters["dbname"]; !exists {
		r.Config.Parameters["dbname"] = r.Datastore
	}
	return nil
}

type DsUnitRegisterResponse struct {
	Modified int
}

type DsUnitPrepareRequest struct {
	Datastore  string
	URL        string
	Credential string
	Prefix     string //apply prefix
	Postfix    string //apply suffix
	Data       map[string][]map[string]interface{}
	Expand bool
}


func (r *DsUnitPrepareRequest) AsDatasetResource() *dsunit.DatasetResource {
	var result = &dsunit.DatasetResource{
		Datastore:  r.Datastore,
		URL:        r.URL,
		Credential: r.Credential,
		Prefix:     r.Prefix,
		Postfix:    r.Postfix,
	}
	if len(r.Data) > 0 {
		result.TableRows = make([]*dsunit.TableRows, 0)
		for table, data := range r.Data {
			var tableRows = &dsunit.TableRows{
				Table: table,
				Rows:  data,
			}
			result.TableRows = append(result.TableRows, tableRows)
		}
	}
	return result
}

func (r *DsUnitPrepareRequest) Validate() error {
	if r.Datastore == "" {
		return fmt.Errorf("Datasets.Datastore was empty")
	}
	if r.URL == "" && len(r.Data) == 0 {
		return fmt.Errorf("Missing data: Datasets.URL/Datasets.TableRows were empty")
	}
	return nil
}

type DsUnitPrepareResponse struct {
	Added    int
	Modified int
	Deleted  int
}

type DsUnitVerifyRequest struct {
	Datasets *dsunit.DatasetResource
	//table to table rows data
	Data        map[string][]map[string]interface{}
	Expand      bool
	CheckPolicy int
}

func (r *DsUnitVerifyRequest) Validate() error {
	if len(r.Data) > 0 && r.Datasets != nil {
		r.Datasets.TableRows = make([]*dsunit.TableRows, 0)
		for table, data := range r.Data {
			var tableRows = &dsunit.TableRows{
				Table: table,
				Rows:  data,
			}
			r.Datasets.TableRows = append(r.Datasets.TableRows, tableRows)
		}
	}
	if r.Datasets == nil {
		return fmt.Errorf("Datasets was nil")
	}
	if r.Datasets.Datastore == "" {
		return fmt.Errorf("Datasets.Datastore was empty")
	}

	if r.Datasets.URL == "" && len(r.Datasets.TableRows) == 0 {
		return fmt.Errorf("Missing data: Datasets.URL/Datasets.TableRows were empty")
	}
	return nil
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
	passwordCredential := &storage.PasswordCredential{}
	err := NewFileResource(credential).JsonDecode(passwordCredential)
	if err != nil {
		return err
	}
	config.Parameters["username"] = passwordCredential.Username
	config.Parameters["password"] = passwordCredential.Password
	config.Init()

	dsManager, err := dsc.NewManagerFactory().Create(config)
	if err != nil {
		return err
	}
	s.Manager.ManagerRegistry().Register(datastoreName, dsManager)
	return nil
}

func (s *dsataStoreUnitService) addMapping(context *Context, request *DsUnitMappingRequest) (*DsUnitMappingResonse, error) {
	var response = &DsUnitMappingResonse{
		Tables: make([]string, 0),
	}
	if request.Mappings != nil {
		for _, mapping := range request.Mappings {
			mappingResource, err := context.ExpandResource(mapping)
			if err != nil {
				return nil, err
			}
			var datasetMapping = &DatasetMappings{}
			err = mappingResource.JsonDecode(datasetMapping)
			if err != nil {
				return nil, err
			}
			response.Tables = append(response.Tables, datasetMapping.Value.Tables()...)
			s.Manager.RegisterDatasetMapping(datasetMapping.Name, datasetMapping.Value)
		}
	}
	return response, nil
}

func (s *dsataStoreUnitService) runScript(context *Context, datastore string, source *Resource) (int, error) {
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
		resource, err := context.ExpandResource(&Resource{URL: datasetResource.URL})
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
				expanded := ExpandValue(row.Values, state)
				row.Values = toolbox.AsMap(expanded)
			}
		}
	}

	response.Added, response.Modified, response.Deleted, err = s.Manager.PrepareDatastore(datasets)
	return response, err
}

func (s *dsataStoreUnitService) verify(context *Context, request *DsUnitVerifyRequest) (interface{}, error) {
	err := request.Validate()
	if err != nil {
		return nil, err
	}
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

	case "mapping":
		return &DsUnitMappingRequest{}, nil
	case "sequence":
		return &DsUnitTableSequenceRequest{}, nil
	case "verify":
		return &DsUnitVerifyRequest{}, nil
	}
	return s.AbstractService.NewRequest(action)
}

func AsTableRecords(source interface{}, state common.Map) (interface{}, error) {
	var result = make(map[string][]map[string]interface{})
	if source == nil {
		return nil, fmt.Errorf("Source was nil")
	}
	if !state.Has(DataStoreUnitServiceId) {
		state.Put(DataStoreUnitServiceId, common.NewMap())
	}
	dataStoreState := state.GetMap(DataStoreUnitServiceId)

	if toolbox.IsSlice(source) {
		for _, item := range toolbox.AsSlice(source) {

			if toolbox.IsMap(item) {
				aMap := toolbox.AsMap(item)
				tableValue, ok := aMap["Table"]
				if !ok {
					return nil, fmt.Errorf("Table was missing in %v", aMap)
				}
				dataValue, ok := aMap["Value"]
				if !ok {
					return nil, fmt.Errorf("Value was missing in %v", aMap)
				}
				if !toolbox.IsMap(dataValue) {
					return nil, fmt.Errorf("Value is not map in %T, %v", dataValue, dataValue)

				}

				value := toolbox.AsMap(ExpandValue(dataValue, state))
				for k, v := range value {
					var textValue = toolbox.AsString(v)
					if strings.HasPrefix(textValue, "$") {
						delete(value, k)
					} else if strings.HasPrefix(textValue, "\\$") {
						value[k] = string(textValue[1:])
					}
				}
				table := toolbox.AsString(tableValue)

				if !dataStoreState.Has(table) {
					dataStoreState.Put(table, common.NewCollection())
				}
				records := dataStoreState.GetCollection(table)
				records.Push(value)

				_, ok = result[table]
				if !ok {
					result[table] = make([]map[string]interface{}, 0)
				}
				result[table] = append(result[table], value)
				autoincrementValue, ok := aMap["Autoincrement"]
				if ok {
					if toolbox.IsSlice(autoincrementValue) {
						for _, key := range toolbox.AsSlice(autoincrementValue) {
							keyText := toolbox.AsString(key)
							value, has := state.GetValue(keyText)
							if !has {
								value = 0
							}
							state.SetValue(keyText, toolbox.AsInt(value)+1)
						}
					}
				}
			}
		}
	}
	return result, nil
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
