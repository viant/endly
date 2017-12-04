package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/storage"
	"github.com/viant/toolbox/url"
	"io"
	"io/ioutil"
	url2 "net/url"
	"path"
	"strings"
)

//TODO refactor compress with https://golangcode.com/create-zip-files-in-go/

//TransferServiceID represents transfer service id
const TransferServiceID = "transfer"

//UseMemoryService flag in the context to ignore
const UseMemoryService = "useMemoryService"

//CopyEventType represents CopyEventType
type CopyEventType struct {
	SourceURL string
	TargetURL string
	Expand    bool
}

type transferService struct {
	*AbstractService
}

//NewExpandedContentHandler return a new reader that can substitude content with state map, replacement data provided in replacement map.
func NewExpandedContentHandler(context *Context, replaceMap map[string]string, expand bool) func(reader io.Reader) (io.Reader, error) {
	return func(reader io.Reader) (io.Reader, error) {
		content, err := ioutil.ReadAll(reader)
		if err != nil {
			return nil, err
		}
		var result = string(content)
		if expand {
			result = context.Expand(result)
			if err != nil {
				return nil, err
			}
		}
		for k, v := range replaceMap {
			result = strings.Replace(result, k, v, len(result))
		}
		return strings.NewReader(toolbox.AsString(result)), nil
	}
}


func (s *transferService) getStorageService(context *Context, resource *url.Resource) (storage.Service, error) {
	if context.state.Has(UseMemoryService) {
		return storage.NewMemoryService(), nil
	}
	return storage.NewServiceForURL(resource.URL, resource.Credential)
}

func (s *transferService) run(context *Context, transfers ...*Transfer) (*TransferCopyResponse, error) {
	var result = &TransferCopyResponse{
		Transferred: make([]*TransferLog, 0),
	}
	for _, transfer := range transfers {
		sourceResource, err := context.ExpandResource(transfer.Source)
		if err != nil {
			return nil, err
		}
		sourceService, err := s.getStorageService(context, sourceResource)
		if err != nil {
			return nil, err
		}
		defer sourceService.Close()
		targetResource, err := context.ExpandResource(transfer.Target)
		if err != nil {
			return nil, err
		}
		targetService, err := s.getStorageService(context, targetResource)
		if err != nil {
			return nil, fmt.Errorf("failed to lookup targetResource storageService for %v: %v", targetResource.URL, err)
		}
		defer targetService.Close()
		var handler func(reader io.Reader) (io.Reader, error)
		if transfer.Expand || len(transfer.Replace) > 0 {
			handler = NewExpandedContentHandler(context, transfer.Replace, transfer.Expand)
		}
		if has, _ := sourceService.Exists(sourceResource.URL); !has {
			return nil, fmt.Errorf("failed to copy: %v %v - Source does not exists", sourceResource.URL, targetResource.URL)
		}

		compressed := transfer.Compress &&
			(sourceResource.ParsedURL.Scheme == "scp" || sourceResource.ParsedURL.Scheme == "file") &&
			(targetResource.ParsedURL.Scheme == "scp" || targetResource.ParsedURL.Scheme == "file")

		var copyEventType = &CopyEventType{
			SourceURL: sourceResource.URL,
			TargetURL: targetResource.URL,
			Expand:    transfer.Expand || len(transfer.Replace) > 0,
		}
		startEvent := s.Begin(context, copyEventType, Pairs("value", copyEventType), Info)
		object, err := sourceService.StorageObject(sourceResource.URL)
		if err != nil {
			return nil, err
		}
		if compressed {
			err = s.compressSource(context, sourceResource, targetResource, object)
			if err != nil {
				return nil, err
			}
		}
		err = storage.Copy(sourceService, sourceResource.URL, targetService, targetResource.URL, handler, nil)
		s.End(context)(startEvent, Pairs())
		if err != nil {
			return result, err
		}
		if compressed {
			err = s.decompressTarget(context, sourceResource, targetResource, object)
			if err != nil {
				return nil, err
			}
		}
		info := NewTransferLog(context, sourceResource.URL, targetResource.URL, err, transfer.Expand)
		result.Transferred = append(result.Transferred, info)
	}
	return result, nil
}



func (s *transferService) compressSource(context *Context, source, target *url.Resource, sourceObject storage.Object) error {
	var parent, name = path.Split(source.ParsedURL.Path)
	var archiveSource = name
	var archiveName = fmt.Sprintf("%v.tar.gz", name)
	if sourceObject.IsFolder() {
		lastDirPosition := strings.LastIndex(source.ParsedURL.Path, "/")
		if lastDirPosition != -1 {
			archiveName = fmt.Sprintf("%v.tar.gz", string(source.ParsedURL.Path[lastDirPosition+1:]))
		}
		archiveSource = "*"
	}

	_, err := context.Execute(source, &CommandRequest{
		Commands: []string{fmt.Sprintf("cd %v", parent)},
	})
	if err == nil {
		_, err = context.Execute(source, &CommandRequest{
			Commands:  []string{fmt.Sprintf("tar cvzf %v %v", archiveName, archiveSource)},
			TimeoutMs: 120000,
		})
	}
	if err != nil {
		return err
	}

	if sourceObject.IsFolder() {
		source.URL = toolbox.URLPathJoin(source.URL, archiveName)
		source.ParsedURL, _ = url2.Parse(source.URL)
		target.URL = toolbox.URLPathJoin(target.URL, archiveName)
		target.ParsedURL, _ = url2.Parse(target.URL)
		return nil
	}

	err = source.Rename(archiveName)
	if err != nil {
		return err
	}
	return target.Rename(archiveName)
}

func (s *transferService) decompressTarget(context *Context, source, target *url.Resource, sourceObject storage.Object) error {
	var parent, name = path.Split(target.ParsedURL.Path)
	_, err := context.Execute(target, &CommandRequest{
		Commands: []string{fmt.Sprintf("mkdir -p %v\ncd %v", parent, parent)},
	})

	if err == nil {
		_, err = context.Execute(target, &CommandRequest{
			Commands:  []string{fmt.Sprintf("tar xvzf %v", name)},
			TimeoutMs: 120000,
		})
	}
	if err == nil && sourceObject.IsFolder() {
		_, err = context.Execute(target, &CommandRequest{
			Commands:  []string{fmt.Sprintf("rm %v", name)},
			TimeoutMs: 120000,
		})
		if err == nil {
			_, err = context.Execute(source, &CommandRequest{
				Commands:  []string{fmt.Sprintf("rm %v", name)},
				TimeoutMs: 120000,
			})
		}
	}

	return err
}

func (s *transferService) Run(context *Context, request interface{}) *ServiceResponse {
	startEvent := s.Begin(context, request, Pairs("request", request))
	var response = &ServiceResponse{Status: "ok"}
	defer s.End(context)(startEvent, Pairs("response", response))
	var err error
	switch actualRequest := request.(type) {
	case *TransferCopyRequest:
		response.Response, err = s.run(context, actualRequest.Transfers...)
		if err != nil {
			response.Error = fmt.Sprintf("failed to tranfer resources: %v, %v", actualRequest.Transfers, err)
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
	case "copy":
		return &TransferCopyRequest{
			Transfers: make([]*Transfer, 0),
		}, nil
	}
	return s.AbstractService.NewRequest(action)
}

//NewTransferService creates a new transfer service
func NewTransferService() Service {
	var result = &transferService{
		AbstractService: NewAbstractService(TransferServiceID),
	}
	result.AbstractService.Service = result
	return result

}
