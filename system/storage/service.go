package storage

import (
	"fmt"
	"github.com/viant/afs"
	"github.com/viant/endly/system/storage/transfer"
	"io"

	"github.com/viant/endly"
	"github.com/viant/toolbox/url"
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

func (s *service) getResourceAndService(context *endly.Context, resource *url.Resource) (*url.Resource, afs.Service, error) {
	expandedResource, err := context.ExpandResource(resource)
	if err != nil {
		return nil, nil, err
	}
	service, err := GetStorageService(context, expandedResource)
	if err != nil {
		return nil, nil, err
	}
	return expandedResource, service, nil
}



func (s *service) getModificationHandler(context *endly.Context, transfer *transfer.Rule) func(reader io.ReadCloser) (io.ReadCloser, error) {
	var handler func(reader io.ReadCloser) (io.ReadCloser, error)
	if transfer.Expand || len(transfer.Replace) > 0 {
		handler = NewExpandedContentHandler(context, transfer.Replace, transfer.Expand)
	}
	return handler
}




func (s *service) registerRoutes() {
	s.Register(&endly.Route{
		Action: "copy",
		RequestInfo: &endly.ActionInfo{
			Description: "transfer content (files or directory structure) from source into destination, both source and destination can use local or remote file system (s3, gs, scp)",
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
		Action: "create",
		RequestInfo: &endly.ActionInfo{
			Description: "create folder or file into target destination, optionally content can be uploaded from sourceKey",
		},
		RequestProvider: func() interface{} {
			return &CreateRequest{}
		},
		ResponseProvider: func() interface{} {
			return &CreateResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*CreateRequest); ok {
				return s.create(context, req)
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

//New creates a new service
func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result

}
