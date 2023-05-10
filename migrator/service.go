package migrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/viant/endly"
)

const ServiceID = "migrator"

const (
	migrateServicePostmanExample = `{
  "Collection": {
    "CollectionPath": "/path/to/collection/file/or/directory",
    "OutputPath": "/path/where/endly/should/write/workflow"
  },
}`
)

type migratorService struct {
	*endly.AbstractService
}

func (s *migratorService) migratePostman(context *endly.Context, request *MigratePostmanRequest) (*MigratePostmanResponse, error) {

	postmanObjects, err := getPostmanObjects(request.CollectionPath)
	if err != nil {
		return nil, err
	}
	builder := convertToRunBuilder(postmanObjects)

	for _, e := range builder.environments {
		s, err := e.TOJson()
		if err != nil {
			return nil, err
		}
		err = os.WriteFile(filepath.Join(request.OutputPath, makeDirOrFileName(e.name)+".json"), []byte(s), 0644)
		if err != nil {
			return nil, err
		}
	}

	requestsPath := filepath.Join(request.OutputPath, "requests")
	err = os.Mkdir(requestsPath, 0750)
	if err != nil {
		return nil, err
	}
	for i, r := range builder.requests {
		s, err := r.TOJson()
		if err != nil {
			return nil, err
		}
		index := strings.TrimLeft(fmt.Sprintf("%03s", fmt.Sprint(i+1)), " ")
		dirName := filepath.Join(requestsPath, index+"_"+makeDirOrFileName(r.name))
		err = os.Mkdir(dirName, 0750)
		if err != nil {
			return nil, err
		}
		err = os.WriteFile(filepath.Join(dirName, "request.json"), []byte(s), 0644)
		if err != nil {
			return nil, err
		}
	}

	defaultPath := filepath.Join(request.OutputPath, "default")
	err = os.Mkdir(defaultPath, 0750)
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(filepath.Join(defaultPath, "send.yaml"), []byte(builder.sendToYaml()), 0644)
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(filepath.Join(request.OutputPath, "run.yaml"), []byte(builder.runToYaml()), 0644)
	if err != nil {
		return nil, err
	}

	return &MigratePostmanResponse{
		OutputPath: request.OutputPath,
		Success:    true,
		Message:    "Success",
	}, nil
}

func (s *migratorService) registerRoutes() {
	s.Register(&endly.Route{
		Action: "postman",
		RequestInfo: &endly.ActionInfo{
			Description: "Migrate postman collection to endly workflow excluding tests v2.1",
			Examples: []*endly.UseCase{
				{
					Description: "migrate postman collection",
					Data:        migrateServicePostmanExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &MigratePostmanRequest{}
		},
		ResponseProvider: func() interface{} {
			return &MigratePostmanResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*MigratePostmanRequest); ok {
				return s.migratePostman(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

func New() endly.Service {
	var result = &migratorService{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
