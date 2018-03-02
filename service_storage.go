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

//TransferServiceID represents transfer service id (legacy name), both name are register with this service
const TransferServiceID = "transfer"

//StorageServiceID represents transfer service id
const StorageServiceID = "storage"

//UseMemoryService flag in the context to ignore
const UseMemoryService = "useMemoryService"

//CompressionTimeout compression/decompression timeout
var CompressionTimeout = 120000

type storageService struct {
	*AbstractService
}

//NewExpandedContentHandler return a new reader that can substitute content with state map, replacement data provided in replacement map.
func NewExpandedContentHandler(context *Context, replaceMap map[string]string, expand bool) func(reader io.ReadCloser) (io.ReadCloser, error) {
	return func(reader io.ReadCloser) (io.ReadCloser, error) {
		defer reader.Close()
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
		return ioutil.NopCloser(strings.NewReader(toolbox.AsString(result))), nil
	}
}

func getStorageService(context *Context, resource *url.Resource) (storage.Service, error) {
	if context.state.Has(UseMemoryService) {
		return storage.NewMemoryService(), nil
	}
	return storage.NewServiceForURL(resource.URL, resource.Credential)
}

//IsShellCompressable returns true if resource can be compress via shell command.
func IsShellCompressable(protScheme string) bool {
	return protScheme == "scp" || protScheme == "file"
}

func (s *storageService) getResourceAndService(context *Context, resource *url.Resource) (*url.Resource, storage.Service, error) {
	expendedResource, err := context.ExpandResource(resource)
	if err != nil {
		return nil, nil, err
	}
	service, err := getStorageService(context, expendedResource)
	if err != nil {
		return nil, nil, err
	}
	return expendedResource, service, nil
}

func (s *storageService) getModificationHandler(context *Context, transfer *Transfer) func(reader io.ReadCloser) (io.ReadCloser, error) {
	var handler func(reader io.ReadCloser) (io.ReadCloser, error)
	if transfer.Expand || len(transfer.Replace) > 0 {
		handler = NewExpandedContentHandler(context, transfer.Replace, transfer.Expand)
	}
	return handler
}

func (s *storageService) compressSource(context *Context, source, target *url.Resource, sourceObject storage.Object) error {
	var baseDirectory, name = path.Split(source.ParsedURL.Path)
	var archiveSource = name

	if sourceObject.IsFolder() {
		baseDirectory = source.DirectoryPath()
		_, name = path.Split(baseDirectory)
		archiveSource = "*"
	}
	var archiveName = fmt.Sprintf("%v.tar.gz", name)
	response, err := context.Execute(source, &CommandRequest{
		Commands: []string{
			fmt.Sprintf("cd %v", baseDirectory),
			fmt.Sprintf("tar cvzf %v %v", archiveName, archiveSource),
		},
		TimeoutMs: CompressionTimeout,
	})

	if err != nil {
		return err
	}
	if CheckNoSuchFileOrDirectory(response.Stdout()) {
		return fmt.Errorf("faied to compress: %v, %v", fmt.Sprintf("tar cvzf %v %v", archiveName, archiveSource), response.Stdout())
	}

	if sourceObject.IsFolder() {
		source.URL = toolbox.URLPathJoin(source.URL, archiveName)
		source.ParsedURL, _ = url2.Parse(source.URL)
		target.URL = toolbox.URLPathJoin(target.URL, archiveName)
		target.ParsedURL, _ = url2.Parse(target.URL)
		return nil
	}

	if err = source.Rename(archiveName); err == nil {
		if path.Ext(target.ParsedURL.Path) != "" {
			err = target.Rename(archiveName)
		} else {
			target.URL = toolbox.URLPathJoin(target.URL, archiveName)
			target.ParsedURL, _ = url2.Parse(target.URL)
		}
	}
	return err
}

func (s *storageService) decompressTarget(context *Context, source, target *url.Resource, sourceObject storage.Object) error {

	var baseDir, name = path.Split(target.ParsedURL.Path)

	_, err := context.Execute(target, &CommandRequest{
		Commands: []string{
			fmt.Sprintf("mkdir -p %v", baseDir),
			fmt.Sprintf("cd %v", baseDir),
		},
	})

	if err == nil {
		_, err = context.Execute(target, &CommandRequest{
			Commands: []string{
				fmt.Sprintf("tar xvzf %v", name),
				fmt.Sprintf("rm %v", name),
			},
			TimeoutMs: CompressionTimeout,
		})
	}
	if err == nil {
		_, err = context.Execute(target, &CommandRequest{
			Commands: []string{
				fmt.Sprintf("cd %v", source.DirectoryPath()),
				fmt.Sprintf("rm %v", name),
			},
		})
	}

	return err
}

func (s *storageService) copy(context *Context, request *StorageCopyRequest) (*StorageCopyResponse, error) {
	var result = &StorageCopyResponse{
		TransferredURL: make([]string, 0),
	}
	for _, transfer := range request.Transfers {
		sourceResource, sourceService, err := s.getResourceAndService(context, transfer.Source)
		if err != nil {
			return nil, err
		}
		defer sourceService.Close()
		targetResource, targetService, err := s.getResourceAndService(context, transfer.Target)
		if err != nil {
			return nil, err
		}
		defer targetService.Close()

		var handler = s.getModificationHandler(context, transfer)
		if has, _ := sourceService.Exists(sourceResource.URL); !has {
			return nil, fmt.Errorf(" %v %v - source does not exists", sourceResource.URL, targetResource.URL)
		}

		compressed := transfer.Compress && IsShellCompressable(sourceResource.ParsedURL.Scheme) && IsShellCompressable(targetResource.ParsedURL.Scheme)
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
		if err != nil {
			return result, err
		}
		if compressed {
			err = s.decompressTarget(context, sourceResource, targetResource, object)
			if err != nil {
				return nil, err
			}
		}
		result.TransferredURL = append(result.TransferredURL, object.URL())
	}
	return result, nil
}

func (s *storageService) remove(context *Context, request *StorageRemoveRequest) (*StorageRemoveResponse, error) {
	var response = &StorageRemoveResponse{
		Removed: make([]string, 0),
	}
	for _, resource := range request.Resources {
		resource, service, err := s.getResourceAndService(context, resource)
		if err != nil {
			return nil, err
		}
		object, err := service.StorageObject(resource.URL)
		if err == nil && object != nil {
			err = service.Delete(object)
			if err != nil {
				return nil, err
			}
			response.Removed = append(response.Removed, resource.URL)
		}
	}
	return response, nil
}

func (s *storageService) download(context *Context, request *StorageDownloadRequest) (*StorageDownloadResponse, error) {
	var response = &StorageDownloadResponse{}
	resource, service, err := s.getResourceAndService(context, request.Source)
	if err != nil {
		return nil, err
	}
	object, err := service.StorageObject(resource.URL)
	if err != nil {
		return nil, err
	}
	if object == nil {
		return nil, fmt.Errorf("failed to lookup resource: %v", resource.URL)
	}
	reader, err := service.Download(object)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if request.Udf != "" {
		response.Transformed, err = TransformWithUDF(context, request.Udf, resource.URL, data)
		if err != nil {
			return nil, err
		}
	}

	response.Payload = AsPayload(data)
	if request.TargetKey != "" {
		var state = context.state
		if response.Transformed != nil {
			state.Put(request.TargetKey, response.Transformed)
		} else {
			state.Put(request.TargetKey, response.Payload)
		}
	}

	return response, nil

}

func (s *storageService) upload(context *Context, request *StorageUploadRequest) (*StorageUploadResponse, error) {
	var response = &StorageUploadResponse{}

	resource, service, err := s.getResourceAndService(context, request.Target)
	if err != nil {
		return nil, err
	}

	var state = context.state
	if !state.Has(request.SourceKey) {
		return nil, fmt.Errorf("sourcekey %v value was empty", request.SourceKey)

	}

	data := state.GetString(request.SourceKey)
	err = service.Upload(resource.URL, strings.NewReader(data))
	if err != nil {
		return nil, err
	}
	response.UploadSize = len(data)
	response.UploadURL = resource.URL
	return response, nil

}

const (
	storageCopySimpleExample = `{
  "Transfers": [
    {
      "Source": {
        "URL": "https://svn.viantinc.com/svn/project/db/schema.ddl",
        "Credential": "${env.HOME}/.secret/svn.json"
      },
      "Target": {
        "URL": "build/db/"
      }
    }
  ]
}`
	storageCopyRemoteTransferExample = `{
  "Transfers": [
    {
      "Source": {
        "URL": "s3://mybucket1/project1/assets/",
        "Credential": "${env.HOME}/.secret/s3.json"
      },
      "Target": {
         "URL": "gs://mybucket2/project1/assets/",
          "Credential": "${env.HOME}/.secret/gs.json"
      }
    }
  ]
}`

	storageCopyReplacementTransferExample = `{
  "Transfers": [
    {
      "Source": {
        "URL": "scp://127.0.0.1/build/app/target/classes/server.properties",
        "Credential": "${env.HOME}/.secret/localhost.json"
      },
      "Target": {
        "URL": "scp://127.0.0.1/build/app/target/target/build/WEB-INF/classes/dserver.properties",
        "Credential": "${env.HOME}/.secret/localhost.json"
      },
      "Replace": {
        "10.2.1.1": "127.0.0.1",
        "xxx.enabled=false": "xxx.enabled=true"
      }
    }
  ]
}`
)

func (s *storageService) registerRoutes() {
	s.Register(&ServiceActionRoute{
		Action: "copy",
		RequestInfo: &ActionInfo{
			Description: "transfer content (files or directory structure) from source into destination, both source and destination can use local or remote file system (s3, gs, scp)",
			Examples: []*ExampleUseCase{
				{
					UseCase: "simple copy",
					Data:    storageCopySimpleExample,
				},
				{
					UseCase: "remote to remote data transfer",
					Data:    storageCopyRemoteTransferExample,
				},
				{
					UseCase: "copy with replacement",
					Data:    storageCopyReplacementTransferExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &StorageCopyRequest{}
		},
		ResponseProvider: func() interface{} {
			return &StorageCopyResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*StorageCopyRequest); ok {
				return s.copy(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "remove",
		RequestInfo: &ActionInfo{
			Description: "remove assets from local or remote file system",
		},
		RequestProvider: func() interface{} {
			return &StorageRemoveRequest{}
		},
		ResponseProvider: func() interface{} {
			return &StorageRemoveResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*StorageRemoveRequest); ok {
				return s.remove(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "upload",
		RequestInfo: &ActionInfo{
			Description: "upload content of state map source key into target destination",
		},
		RequestProvider: func() interface{} {
			return &StorageUploadRequest{}
		},
		ResponseProvider: func() interface{} {
			return &StorageUploadResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*StorageUploadRequest); ok {
				return s.upload(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&ServiceActionRoute{
		Action: "download",
		RequestInfo: &ActionInfo{
			Description: "download content from source into state map key",
		},
		RequestProvider: func() interface{} {
			return &StorageDownloadRequest{}
		},
		ResponseProvider: func() interface{} {
			return &StorageDownloadResponse{}
		},
		Handler: func(context *Context, request interface{}) (interface{}, error) {
			if handlerRequest, ok := request.(*StorageDownloadRequest); ok {
				return s.download(context, handlerRequest)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//NewStorageService creates a new transfer service
func NewStorageService() Service {
	var result = &storageService{
		AbstractService: NewAbstractService(StorageServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result

}
