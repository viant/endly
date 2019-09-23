package storage

import (
	"errors"
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/system/storage/transfer"
	"github.com/viant/endly/udf"
	"github.com/viant/toolbox/url"
)

//CopyRequest represents a resources copy request
type CopyRequest struct {
	*transfer.Rule `description:"if asset uses relative path it will be joined with this URL" json:",inline"`
	Assets             transfer.Assets  `description:"map entry can either represent a transfer struct or simple key is the source and the value destination relative path"` // transfers
	Transfers          []*transfer.Rule `description:"actual transfer assets, if empty it derives from assets or source/desc "`
	Udf                string               `description:"custom user defined function to returns a CopyHandler type func which performs the copy"`
}


//CopyResponse represents a resources copy response
type CopyResponse struct {
	TransferredURL []string //transferred URLs
}



func (s *service) copy(context *endly.Context, request *CopyRequest) (*CopyResponse, error) {
	var result = &CopyResponse{
		TransferredURL: make([]string, 0),
	}

	for _, rule := range request.Transfers {
		sourceResource, sourceService, err := s.getResourceAndService(context, rule.Source)
		if err != nil {
			return nil, err
		}


		defer sourceService.Close()
		destResource, destService, err := s.getResourceAndService(context, rule.Dest)
		if err != nil {
			return nil, err
		}
		defer destService.Close()

		var handler = s.getModificationHandler(context, rule)

		if has, _ := sourceService.Exists(sourceResource.URL); !has {
			return nil, fmt.Errorf(" %v %v - source does not exists (%T)", sourceResource.URL, destResource.URL, sourceService)
		}

		useCompression := rule.Compress && IsShellCompressable(sourceResource.ParsedURL.Scheme) && IsShellCompressable(destResource.ParsedURL.Scheme)
		object, err := sourceService.StorageObject(sourceResource.URL)
		if err != nil {
			return nil, err
		}

		if useCompression {
			err = s.compressSource(context, sourceResource, destResource, object)
			if err != nil {
				return nil, err
			}
		}


		// Custom CopyHandler wrapped as UDFs
		var copyHandler storage.CopyHandler
		if request.Udf != "" && object.IsContent() {
			udf, err := udf.TransformWithUDF(context, request.Udf, rule.Source.URL, nil)
			if err != nil {
				return nil, err
			}
			copyHandler = udf.(storage.CopyHandler)
		}

		err = storage.Copy(sourceService, sourceResource.URL, destService, destResource.URL, handler, copyHandler)
		if err != nil {
			return result, err
		}
		if useCompression {
			err = s.decompressTarget(context, sourceResource, destResource, object)
			if err != nil {
				return nil, err
			}
		}
		result.RulerredURL = append(result.RulerredURL, object.URL())
	}
	return result, nil
}



//CopyRequest creates a new copy request
func NewCopyRequest(assets transfer.Assets, transfers ...*transfer.Rule) *CopyRequest {
	var super *transfer.Rule
	if len(transfers) > 0 {
		super = transfers[0]
		transfers = transfers[1:]
	}
	return &CopyRequest{
		Rule:  super,
		Assets:    assets,
		Transfers: transfers,
	}
}

//NewCopyRequestFromuRL creates a new request from URL (JSON or YAML format are supported)
func NewCopyRequestFromuRL(URL string) (*CopyRequest, error) {
	var request = &CopyRequest{}
	resource := url.NewResource(URL)
	if err := resource.Decode(request); err != nil {
		return nil, err
	}
	return request, nil
}

//Init initialises request
func (r *CopyRequest) Init() error {
	if r.Rule == nil {
		r.Rule = &transfer.Rule{}
	} else {
		if err := r.Rule.Init(); err != nil {
			return err
		}
	}

	hasAssets := len(r.Assets) > 0
	hasTransfers := len(r.Transfers) > 0
	if hasTransfers {
		if r.Source == nil && r.Dest == nil {
			return nil
		}

		for _, rule := range r.Transfers {
			if rule.Source != nil {
				if r.Source == nil {
					r.Source = url.NewResource("/")
				}
				rule.Source = transfer.JoinIfNeeded(r.Source, rule.Source.URL)
			}
			if rule.Dest != nil {
				rule.Dest = transfer.JoinIfNeeded(r.Dest, rule.Dest.URL)
			}
		}
		return nil
	}

	if !hasAssets {
		if r.Source != nil && r.Dest != nil {
			r.Transfers = []*transfer.Rule{
				r.Rule,
			}
		}
		return nil
	}

	r.Transfers = r.Assets.AsTransfer(r.Rule)
	return nil
}

//Validate checks if request is valid
func (r *CopyRequest) Validate() error {
	if len(r.Transfers) == 0 {
		return errors.New("transfers were empty")
	}
	for _, rule := range r.Transfers {
		if err := rule.Validate(); err != nil {
			return err
		}
	}
	return nil
}
