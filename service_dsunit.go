package endly

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/viant/dsc"
	"github.com/viant/dsunit"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
	"strings"
)

const DataStoreUnitServiceId = "dsunit"

type DsUnitRegisterRequest struct {
	Datastore       string
	Config          *dsc.Config //make sure Deploy.Parameters have database Id key
	Credential      string
	adminConfig     *dsc.Config //make sure Deploy.Parameters have database Id key
	AdminDatastore  string      //Id of admin db
	AdminCredential string
	ClearDatastore  bool
	Scripts         []*url.Resource
}

//DatasetMapping represnts dataset mappings
type DatasetMappings struct {
	Name  string                 //mapping Id
	Value *dsunit.DatasetMapping //actual mappings
}

type DsUnitMappingRequest struct {
	Mappings []*url.Resource
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
	Expand     bool
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

type DsUnitPrepareTableData struct {
	Table         string
	Value         interface{}
	AutoGenerate  map[string]string `json:",omitempty"`
	PostIncrement []string          `json:",omitempty"`
	Key           string
}

func (d *DsUnitPrepareTableData) AuotGenerateIfNeeded(state data.Map) error {
	for k, v := range d.AutoGenerate {
		value, has := state.GetValue(v)
		if !has {
			return fmt.Errorf("Failed to autogenerate value for %v - unable to eval: %v \n", k, v)
		}
		state.SetValue(k, value)
	}
	return nil
}

func (d *DsUnitPrepareTableData) PostIncrementIfNeeded(state data.Map) {
	for _, key := range d.PostIncrement {
		keyText := toolbox.AsString(key)
		value, has := state.GetValue(keyText)
		if !has {
			value = 0
		}
		state.SetValue(keyText, toolbox.AsInt(value)+1)
	}
}

func (d *DsUnitPrepareTableData) GetValues(state data.Map) []map[string]interface{} {
	if toolbox.IsMap(d.Value) {
		return []map[string]interface{}{
			d.GetValue(state, d.Value),
		}
	}
	var result = make([]map[string]interface{}, 0)
	if toolbox.IsSlice(d.Value) {
		var aSlice = toolbox.AsSlice(d.Value)
		for _, item := range aSlice {
			value := d.GetValue(state, item)
			result = append(result, value)
		}
	}
	return result
}

func (d *DsUnitPrepareTableData) expandThis(textValue string, value map[string]interface{}) interface{} {
	if strings.Contains(textValue, "this.") {
		var thisState = data.NewMap()
		for subKey, subValue := range value {
			if toolbox.IsString(subValue) {
				subKeyTextValue := toolbox.AsString(subValue)
				if !strings.Contains(subKeyTextValue, "this") {
					thisState.SetValue(fmt.Sprintf("this.%v", subKey), subKeyTextValue)
				}
			}
		}
		return thisState.Expand(textValue)
	}
	return textValue
}

func (d *DsUnitPrepareTableData) GetValue(state data.Map, source interface{}) map[string]interface{} {
	value := toolbox.AsMap(state.Expand(source))
	for k, v := range value {
		var textValue = toolbox.AsString(v)
		if strings.Contains(textValue, "this") {
			value[k] = d.expandThis(textValue, value)
		} else if strings.HasPrefix(textValue, "$") {
			delete(value, k)
		} else if strings.HasPrefix(textValue, "\\$") {
			value[k] = string(textValue[1:])
		}
	}

	dataStoreState := state.GetMap(DataStoreUnitServiceId)

	var key = d.Key
	if key == "" {
		key = d.Table
	}
	if !dataStoreState.Has(key) {
		dataStoreState.Put(key, data.NewCollection())
	}
	records := dataStoreState.GetCollection(key)
	records.Push(value)
	return value
}

func AsTableRecords(source interface{}, state data.Map) (interface{}, error) {
	var result = make(map[string][]map[string]interface{})
	if source == nil {
		return nil, reportError(fmt.Errorf("Source was nil"))
	}
	if !state.Has(DataStoreUnitServiceId) {
		state.Put(DataStoreUnitServiceId, data.NewMap())
	}

	var prepareTableData = []*DsUnitPrepareTableData{}
	err := converter.AssignConverted(&prepareTableData, source)
	if err != nil {
		return nil, err
	}
	for _, tableData := range prepareTableData {
		var table = tableData.Table
		err = tableData.AuotGenerateIfNeeded(state)
		if err != nil {
			return nil, err
		}
		result[table] = append(result[table], tableData.GetValues(state)...)
		tableData.PostIncrementIfNeeded(state)
	}
	dataStoreState := state.GetMap(DataStoreUnitServiceId)
	var variable = &Variable{
		Name:    DataStoreUnitServiceId,
		Persist: true,
		Value:   dataStoreState,
	}
	err = variable.PersistValue()
	if err != nil {
		return nil, err
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
