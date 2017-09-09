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
	Source string
	Target string
	Error string
	Parsable string
	State common.Map
}

func NewTransferInfo(context *Context, source, target string, err error, parsable bool) *TransferInfo {
	result := &TransferInfo{
		Source:source,
		Target:target,
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
	debug := context.Debug()
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
		debug.Log(info)
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

func (s *transferService) Run(context *Context, request interface{}) *Response {
	var response = &Response{Status: "ok"}
	switch actualRequest := request.(type) {
	case *TransfersRequest:
		response.Response, response.Error = s.run(context, actualRequest.Transfers...)
	case *TransferRequest:
		response.Response, response.Error = s.run(context, actualRequest)

	default:
		response.Error = fmt.Errorf("Unsupported request type: %T", request)
	}
	if response.Error != nil {
		response.Status = "err"
	}
	return response
}

func (s *transferService) NewRequest(name string) (interface{}, error) {
	switch name {
	case "run":
		return &TransfersRequest{
			Transfers: make([]*TransferRequest, 0),
		}, nil
	}
	return nil, fmt.Errorf("Unsupported name: %v", name)
}

func NewTransferService() Service {
	var result = &transferService{
		AbstractService: NewAbstractService(TransferServiceId),
	}
	result.AbstractService.Service = result
	return result

}
