package endly

import (
	"fmt"
	"github.com/viant/endly/common"
	"github.com/viant/toolbox/storage"
	_ "github.com/viant/toolbox/storage/scp"
	"io"
)

const TransferServiceId = "transfer"

type TransfersRequest struct {
	Transfers []*TransferRequest
}

type TransferRequest struct {
	Source   *Resource
	Target   *Resource
	Parsable bool
}

type TransferInfo struct {
	Source   string
	Target   string
	Error    string
	Parsable string
	State    common.Map
}

func NewTransferInfo(context *Context, source, target string, err error, parsable bool) *TransferInfo {
	result := &TransferInfo{
		Source: source,
		Target: target,
	}
	if parsable {
		var state = context.State()
		result.State = state.Clone()
	}
	if err != nil {
		result.Error = fmt.Sprintf("%v", err)
	}
	return result
}

type transferService struct {
	*AbstractService
}

func (s *transferService) run(context *Context, transfers ...*TransferRequest) ([]*TransferInfo, error) {
	var result = make([]*TransferInfo, 0)
	sessionInfo := context.SessionInfo()
	for _, transfer := range transfers {

		source, err := context.ExpandResource(transfer.Source)
		if err != nil {
			return nil, err
		}
		sourceService, err := storage.NewServiceForURL(source.URL, source.CredentialFile)
		if err != nil {
			return nil, err
		}
		target, err := context.ExpandResource(transfer.Target)
		if err != nil {
			return nil, err
		}
		targetService, err := storage.NewServiceForURL(target.URL, target.CredentialFile)
		if err != nil {
			return nil, fmt.Errorf("Failed to lookup target storageService for %v: %v", target.URL, err)
		}
		var handler func(reader io.Reader) (io.Reader, error)
		if transfer.Parsable {
			handler = common.NewHandler(context.Context)
		}
		err = storage.Copy(sourceService, source.URL, targetService, target.URL, handler)
		info := NewTransferInfo(context, source.URL, target.URL, err, transfer.Parsable)
		sessionInfo.Log(info)
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

func (s *transferService) Run(context *Context, request interface{}) *ServiceResponse {
	var response = &ServiceResponse{Status: "ok"}
	var err error
	switch actualRequest := request.(type) {
	case *TransfersRequest:
		response.Response, err = s.run(context, actualRequest.Transfers...)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to tranfer resources: %v, %v", actualRequest.Transfers, err)
		}
	case *TransferRequest:
		response.Response, err = s.run(context, actualRequest)
		if err != nil {
			response.Error = fmt.Sprintf("Failed to tranfer resources: %v, %v", actualRequest, err)
		}

	default:
		response.Error = fmt.Sprintf("Unsupported request type: %T", request)
	}
	if response.Error != "" {
		response.Status = "err"
	}
	return response
}

func (s *transferService) NewRequest(action string) (interface{}, error) {
	switch action {
	case "run":
		return &TransfersRequest{
			Transfers: make([]*TransferRequest, 0),
		}, nil
	}
	return s.AbstractService.NewRequest(action)
}

func NewTransferService() Service {
	var result = &transferService{
		AbstractService: NewAbstractService(TransferServiceId),
	}
	result.AbstractService.Service = result
	return result

}
