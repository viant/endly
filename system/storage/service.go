package storage

import (
	"fmt"
	"github.com/viant/endly"
)


const (
	//ServiceID represents transfer service id
	ServiceID = "storage"

	//useMemoryService flag in the context to ignore
	useMemoryService   = "useMemoryService"
	//compressionTimeoutMs max SSH execution time before timing out
	compressionTimeoutMs = 120000
)


type service struct {
	*endly.AbstractService
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
				return s.Copy(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "remove",
		RequestInfo: &endly.ActionInfo{
			Description: "Remove Transfers from local or remote file system",
		},
		RequestProvider: func() interface{} {
			return &RemoveRequest{}
		},
		ResponseProvider: func() interface{} {
			return &RemoveResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*RemoveRequest); ok {
				return s.Remove(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "upload",
		RequestInfo: &endly.ActionInfo{
			Description: "Upload content of state map source key into target destination",
		},
		RequestProvider: func() interface{} {
			return &UploadRequest{}
		},
		ResponseProvider: func() interface{} {
			return &UploadResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*UploadRequest); ok {
				return s.Upload(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "create",
		RequestInfo: &endly.ActionInfo{
			Description: "Create folder or file into target destination, optionally content can be uploaded from sourceKey",
		},
		RequestProvider: func() interface{} {
			return &CreateRequest{}
		},
		ResponseProvider: func() interface{} {
			return &CreateResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*CreateRequest); ok {
				return s.Create(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "download",
		RequestInfo: &endly.ActionInfo{
			Description: "Download content from source into state map key",
		},
		RequestProvider: func() interface{} {
			return &DownloadRequest{}
		},
		ResponseProvider: func() interface{} {
			return &DownloadResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*DownloadRequest); ok {
				return s.Download(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "exists",
		RequestInfo: &endly.ActionInfo{
			Description: "Check if supplied asset exists",
		},
		RequestProvider: func() interface{} {
			return &ExistsRequest{}
		},
		ResponseProvider: func() interface{} {
			return &ExistsResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*ExistsRequest); ok {
				return s.Exists(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "list",
		RequestInfo: &endly.ActionInfo{
			Description: "List supplied location",
		},
		RequestProvider: func() interface{} {
			return &ListRequest{}
		},
		ResponseProvider: func() interface{} {
			return &ListResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*ListRequest); ok {
				return s.List(context, req)
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
