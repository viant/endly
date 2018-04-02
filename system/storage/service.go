package storage

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/system/exec"
	"github.com/viant/endly/udf"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/storage"
	"github.com/viant/toolbox/url"
	"io"
	"io/ioutil"
	url2 "net/url"
	"path"
	"strings"
)

//ServiceID represents transfer service id
const ServiceID = "storage"

//useMemoryService flag in the context to ignore
const useMemoryService = "useMemoryService"

//CompressionTimeout compression/decompression timeout
var CompressionTimeout = 120000

type service struct {
	*endly.AbstractService
}

func (s *service) getResourceAndService(context *endly.Context, resource *url.Resource) (*url.Resource, storage.Service, error) {
	expendedResource, err := context.ExpandResource(resource)
	if err != nil {
		return nil, nil, err
	}
	service, err := GetStorageService(context, expendedResource)
	if err != nil {
		return nil, nil, err
	}
	return expendedResource, service, nil
}

func (s *service) getModificationHandler(context *endly.Context, transfer *Transfer) func(reader io.ReadCloser) (io.ReadCloser, error) {
	var handler func(reader io.ReadCloser) (io.ReadCloser, error)
	if transfer.Expand || len(transfer.Replace) > 0 {
		handler = NewExpandedContentHandler(context, transfer.Replace, transfer.Expand)
	}
	return handler
}

func (s *service) compressSource(context *endly.Context, source, target *url.Resource, sourceObject storage.Object) (err error) {
	var baseDirectory, name = path.Split(source.ParsedURL.Path)
	var archiveSource = name

	if sourceObject.IsFolder() {
		baseDirectory = source.DirectoryPath()
		_, name = path.Split(baseDirectory)
		archiveSource = "."
	}
	var archiveName = fmt.Sprintf("%v.tar.gz", name)

	var runRequest = exec.NewRunRequest(source, false,
		fmt.Sprintf("cd %v", baseDirectory),
		fmt.Sprintf("tar cvzf %v %v", archiveName, archiveSource),
	)
	runRequest.TimeoutMs = CompressionTimeout
	runResponse := &exec.RunResponse{}
	if err = endly.Run(context, runRequest, runResponse); err != nil {
		return err
	}
	if util.CheckNoSuchFileOrDirectory(runResponse.Stdout()) {
		return fmt.Errorf("faied to compress: %v, %v", fmt.Sprintf("tar cvzf %v %v", archiveName, archiveSource), runResponse.Stdout())
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

func (s *service) decompressTarget(context *endly.Context, source, target *url.Resource, sourceObject storage.Object) error {
	var baseDir, name = path.Split(target.ParsedURL.Path)
	var runRequest = exec.NewRunRequest(target, false,
		fmt.Sprintf("mkdir -p %v", baseDir),
		fmt.Sprintf("cd %v", baseDir),
		fmt.Sprintf("tar xvzf %v", name),
		fmt.Sprintf("rm %v", name),
		fmt.Sprintf("cd %v", source.DirectoryPath()))
	runRequest.TimeoutMs = CompressionTimeout
	return endly.Run(context, runRequest, nil)
}

func (s *service) copy(context *endly.Context, request *CopyRequest) (*CopyResponse, error) {
	var result = &CopyResponse{
		TransferredURL: make([]string, 0),
	}
	for _, transfer := range request.Transfers {
		sourceResource, sourceService, err := s.getResourceAndService(context, transfer.Source)
		if err != nil {
			return nil, err
		}
		defer sourceService.Close()
		targetResource, targetService, err := s.getResourceAndService(context, transfer.Dest)
		if err != nil {
			return nil, err
		}
		defer targetService.Close()

		var handler = s.getModificationHandler(context, transfer)

		if has, _ := sourceService.Exists(sourceResource.URL); !has {
			return nil, fmt.Errorf(" %v %v - source does not exists (%T)", sourceResource.URL, targetResource.URL, sourceService)
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

func (s *service) remove(context *endly.Context, request *RemoveRequest) (*RemoveResponse, error) {
	var response = &RemoveResponse{
		Removed: make([]string, 0),
	}
	for _, resource := range request.Assets {
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

func (s *service) download(context *endly.Context, request *DownloadRequest) (*DownloadResponse, error) {
	var response = &DownloadResponse{}
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
		response.Transformed, err = udf.TransformWithUDF(context, request.Udf, resource.URL, data)
		if err != nil {
			return nil, err
		}
	}

	response.Payload = util.AsPayload(data)
	if request.DestKey != "" {
		var state = context.State()
		if response.Transformed != nil {
			state.Put(request.DestKey, response.Transformed)
		} else {
			state.Put(request.DestKey, response.Payload)
		}
	}

	return response, nil

}

func (s *service) upload(context *endly.Context, request *UploadRequest) (*UploadResponse, error) {
	var response = &UploadResponse{}

	resource, service, err := s.getResourceAndService(context, request.Dest)
	if err != nil {
		return nil, err
	}

	var state = context.State()
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
      "Source": {
        "URL": "https://svn.viantinc.com/svn/project/db/schema.ddl",
        "Credentials": "${env.HOME}/.secret/svn.json"
      },
      "Desc": {
        "URL": "build/db/"
      }
}`
	storageCopyRemoteTransferExample = `{
  "Transfers": [
    {
      "Source": {
        "URL": "s3://mybucket1/project1/Transfers/",
        "Credentials": "${env.HOME}/.secret/s3.json"
      },
      "Desc": {
         "URL": "gs://mybucket2/project1/Transfers/",
          "Credentials": "${env.HOME}/.secret/gs.json"
      }
    }
  ]
}`

	storageBatchCopyTransferExample = `{
	"Source": {
		"URL": "s3://mybucket1/",
		"Credentials": "${env.HOME}/.secret/s3.json"
    },
	"Dest": {
		 "URL": "gs://mybucket2/",
		  "Credentials": "${env.HOME}/.secret/gs.json"
    },
	"Assets":{
		"project1/data/":"archive/data/",
		"project1/config/":"setting/config/"
	}
}`

	storageCopyReplacementTransferExample = `{
  "Transfers": [
    {
      "Source": {
        "URL": "scp://127.0.0.1/build/app/target/classes/server.properties",
        "Credentials": "${env.HOME}/.secret/localhost.json"
      },
      "Desc": {
        "URL": "scp://127.0.0.1/build/app/target/target/build/WEB-INF/classes/dserver.properties",
        "Credentials": "${env.HOME}/.secret/localhost.json"
      },
      "Replace": {
        "10.2.1.1": "127.0.0.1",
        "xxx.enabled=false": "xxx.enabled=true"
      }
    }
  ]
}`
)

func (s *service) registerRoutes() {
	s.Register(&endly.Route{
		Action: "copy",
		RequestInfo: &endly.ActionInfo{
			Description: "transfer content (files or directory structure) from source into destination, both source and destination can use local or remote file system (s3, gs, scp)",
			Examples: []*endly.UseCase{
				{
					Description: "simple copy",
					Data:        storageCopySimpleExample,
				},
				{
					Description: "remote to remote data transfer",
					Data:        storageCopyRemoteTransferExample,
				},
				{
					Description: "copy with replacement",
					Data:        storageCopyReplacementTransferExample,
				},
				{
					Description: "batch coopy",
					Data:        storageBatchCopyTransferExample,
				},
			},
		},
		RequestProvider: func() interface{} {
			return &CopyRequest{}
		},
		ResponseProvider: func() interface{} {
			return &CopyResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*CopyRequest); ok {
				return s.copy(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "remove",
		RequestInfo: &endly.ActionInfo{
			Description: "remove Transfers from local or remote file system",
		},
		RequestProvider: func() interface{} {
			return &RemoveRequest{}
		},
		ResponseProvider: func() interface{} {
			return &RemoveResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*RemoveRequest); ok {
				return s.remove(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "upload",
		RequestInfo: &endly.ActionInfo{
			Description: "upload content of state map source key into target destination",
		},
		RequestProvider: func() interface{} {
			return &UploadRequest{}
		},
		ResponseProvider: func() interface{} {
			return &UploadResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*UploadRequest); ok {
				return s.upload(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "download",
		RequestInfo: &endly.ActionInfo{
			Description: "download content from source into state map key",
		},
		RequestProvider: func() interface{} {
			return &DownloadRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DownloadResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*DownloadRequest); ok {
				return s.download(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

//New creates a new storage service
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result

}
