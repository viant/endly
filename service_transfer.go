package endly

import (
	"fmt"
	"github.com/viant/endly/common"
	"github.com/viant/toolbox/storage"
	_ "github.com/viant/toolbox/storage/scp"
	"io"
)

const TransferServiceId = "transfer"

type Transfers struct {
	Transfers []*Transfer
}

type Transfer struct {
	Source   *Resource
	Target   *Resource
	Parsable bool
}

type transferService struct {
	*AbstractService
}

func (s *transferService) run(context *Context, transfers ...*Transfer) (interface{}, error) {
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
		err = storage.Copy(sourceService, transfer.Source.URL, targetService, target.URL, handler)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (s *transferService) Run(context *Context, request interface{}) *Response {
	var response = &Response{Status: "ok"}
	switch typeRequest := request.(type) {
	case *Transfers:
		response.Response, response.Error = s.run(context, typeRequest.Transfers...)
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
		return &Transfers{
			Transfers: make([]*Transfer, 0),
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
