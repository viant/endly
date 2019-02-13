package webdb

import (
	"fmt"
	"github.com/viant/dsc"
	"net/http"
)

const (
	getAllDummySQL     = "SELECT id, name, type_id FROM dummy"
	findDumySQL        = getAllDummySQL + " WHERE id = ?"
	getAllDummyTypeSQL = "SELECT id, name FROM dummy_type"
)

//Service represents application service
type Service struct {
	dsc.Manager
	DbConfig *dsc.Config
}

//GetAll returns all dummy types
func (s *Service) GetAllTypes(request *GetTypeRequest) *GetTypeResponse {
	var response = &GetTypeResponse{Response: NewResponse(), Data: []*DummyType{}}
	err := s.ReadAll(&response.Data, getAllDummyTypeSQL, nil, nil)
	if err != nil {
		response.SetError(err)
		return response
	}
	return response
}

func indexDummyType(response *GetTypeResponse) map[int]*DummyType {
	if response.Status != "ok" {
		return map[int]*DummyType{}
	}
	var result = map[int]*DummyType{}
	for _, dummyType := range response.Data {
		result[dummyType.Id] = dummyType
	}
	return result
}

//GetAll returns all dummy
func (s *Service) GetAll(request *GetRequest) *GetResponse {
	var response = &GetResponse{Response: NewResponse(), Data: []*Dummy{}}
	indexedType := indexDummyType(s.GetAllTypes(&GetTypeRequest{}))
	err := s.ReadAllWithHandler(getAllDummySQL, nil, func(scanner dsc.Scanner) (toContinue bool, err error) {
		var dummy = &Dummy{}
		var typeId int
		if err = scanner.Scan(&dummy.Id, &dummy.Name, &typeId); err != nil {
			return false, err
		}
		dummy.Type = indexedType[typeId]
		response.Data = append(response.Data, dummy)
		return true, nil
	})
	if err != nil {
		response.SetError(err)
		return response
	}
	return response
}

//Find returns data for supplied id
func (s *Service) Find(request *FindRequest) *FindResponse {
	var response = &FindResponse{Response: NewResponse(), Data: &Dummy{}}
	has, err := s.ReadSingle(response.Data, findDumySQL, []interface{}{request.Id}, nil)
	if err != nil {
		response.SetError(err)
		return response
	}
	if !has {
		code := http.StatusNotFound
		response.StatusCode = &code
	}
	return response
}

//Persist persist supplied data
func (s *Service) Persist(request *PersistRequest) *PersistResponse {
	var response = &PersistResponse{Response: NewResponse(), Data: &Dummy{}}
	if request.Data == nil {
		response.SetError(fmt.Errorf("data was empty"))
		return response
	}
	_, _, err := s.Manager.PersistSingle(request.Data, "dummy", nil)
	if err != nil {
		response.SetError(err)
		return response
	}
	response.Data = request.Data
	return response
}

//Handle handles request
func (s *Service) Handle(request *http.Request, writer http.ResponseWriter) {
	writer.WriteHeader(http.StatusOK)
}

//New creates a new service
func New(config *dsc.Config) (*Service, error) {
	manager, err := dsc.NewManagerFactory().Create(config)
	if err != nil {
		return nil, err
	}
	return &Service{Manager: manager, DbConfig: config}, nil
}
