package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/storage"
	"github.com/viant/toolbox/url"
	"io"
	"io/ioutil"
	"path"
	"strings"
	url2 "net/url"
)

//TODO refactor compress with https://golangcode.com/create-zip-files-in-go/


//TransferServiceID represents transfer service id
const TransferServiceID = "transfer"

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

func (s *transferService) run(context *Context, transfers ...*Transfer) (*TransferCopyResponse, error) {
	var result = &TransferCopyResponse{
		Transferred: make([]*TransferLog, 0),
	}
	for _, transfer := range transfers {
		source, err := context.ExpandResource(transfer.Source)
		if err != nil {
			return nil, err
		}
		sourceService, err := storage.NewServiceForURL(source.URL, source.Credential)
		if err != nil {
			return nil, err
		}
		defer sourceService.Close()
		target, err := context.ExpandResource(transfer.Target)
		if err != nil {
			return nil, err
		}
		targetService, err := storage.NewServiceForURL(target.URL, target.Credential)
		if err != nil {
			return nil, fmt.Errorf("Failed to lookup target storageService for %v: %v", target.URL, err)
		}
		defer targetService.Close()
		var handler func(reader io.Reader) (io.Reader, error)
		if transfer.Expand || len(transfer.Replace) > 0 {
			handler = NewExpandedContentHandler(context, transfer.Replace, transfer.Expand)
		}
		if _, err := sourceService.StorageObject(source.URL); err != nil {
			return nil, fmt.Errorf("Failed to copy: %v %v - Source does not exists", source.URL, target.URL)
		}

		compressed := transfer.Compress &&
			(source.ParsedURL.Scheme == "scp" || source.ParsedURL.Scheme == "file") &&
			(target.ParsedURL.Scheme == "scp" || target.ParsedURL.Scheme == "file")


		var copyEventType = &CopyEventType{
			SourceURL: source.URL,
			TargetURL: target.URL,
			Expand:    transfer.Expand || len(transfer.Replace) > 0,
		}
		startEvent := s.Begin(context, copyEventType, Pairs("value", copyEventType), Info)

		if compressed {
			err = s.compressSource(context, source, target)
			if err != nil {
				return nil, err
			}
		}
		err = storage.Copy(sourceService, source.URL, targetService, target.URL, handler, nil)
		s.End(context)(startEvent, Pairs())
		if err != nil {
			return result, err
		}
		if compressed {
			err = s.decompressTarget(context, target)
			if err != nil {
				return nil, err
			}
		}
		info := NewTransferLog(context, source.URL, target.URL, err, transfer.Expand)
		result.Transferred = append(result.Transferred, info)

	}
	return result, nil
}

func (s *transferService) 	compressSource(context *Context, source, target *url.Resource) error {
	var parent, name = path.Split(source.ParsedURL.Path)
	var archiveSource = name
	var archiveName = fmt.Sprintf("%v.gz", name)
	if name == "" {
		lastDirPosition := strings.LastIndex(source.ParsedURL.Path, "/")
		if lastDirPosition != -1 {
			archiveName = string(source.ParsedURL.Path[lastDirPosition+1:])
		}
		archiveSource = "*"
	}

	_, err := context.Execute(source, fmt.Sprintf("cd %v\ntar cvzf %v %v", parent, archiveName, archiveSource))
	if err != nil {
		return err
	}
	if name == "" {
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
	_, name = toolbox.URLSplit(target.URL)
	var targetName = fmt.Sprintf("%v.gz", name)
	return source.Rename(targetName)
}

func (s *transferService) decompressTarget(context *Context, target *url.Resource) error {
	var parent, name = path.Split(target.ParsedURL.Path)
	_, err := context.Execute(target, fmt.Sprintf("cd %v\ntar xvzf %v", parent, name))
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
			response.Error = fmt.Sprintf("Failed to tranfer resources: %v, %v", actualRequest.Transfers, err)
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
