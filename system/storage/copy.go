package storage

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/viant/afs/option"
	"github.com/viant/endly"
	"github.com/viant/endly/system/storage/copy"
	"github.com/viant/endly/udf"
	"github.com/viant/toolbox/url"
)

// CopyRequest represents a resources Copy request
type CopyRequest struct {
	*copy.Rule `description:"if asset uses relative path it will be joined with this URL" json:",inline"`
	Assets     copy.Assets  `description:"map entry can either represent a transfer struct or simple key is the source and the value destination relative path"` // transfers
	Transfers  []*copy.Rule `description:"actual transfer assets, if empty it derives from assets or source/desc "`
	Udf        string       `description:"custom user defined function to return github.com/viant/afs/option.Modifier type to modify copied content"`
}

// CopyResponse represents a resources Copy response
type CopyResponse struct {
	URLs []string //transferred URLs
}

// Copy copy source to dest
func (s *service) Copy(context *endly.Context, request *CopyRequest) (*CopyResponse, error) {
	var response = &CopyResponse{
		URLs: make([]string, 0),
	}
	return response, s.copy(context, request, response)
}

func (s *service) copy(context *endly.Context, request *CopyRequest, response *CopyResponse) error {
	var udfModifier option.Modifier
	if request.Udf != "" {
		var ok bool
		UDF, err := udf.TransformWithUDF(context, request.Udf, "", nil)
		if err != nil {
			return errors.Wrapf(err, "failed to get udf: %v", request.Udf)
		}
		udfModifier, ok = UDF.(option.Modifier)
		if !ok {
			return fmt.Errorf("udf %v does not implement %T", UDF, udfModifier)
		}
	}
	for _, rule := range request.Transfers {
		if err := s.transfer(context, rule, udfModifier, response); err != nil {
			return err
		}
	}
	return nil
}

func (s *service) transfer(context *endly.Context, rule *copy.Rule, udfModifier option.Modifier, response *CopyResponse) error {
	source, sourceOpts, err := getSourceWithOptions(context, rule)
	if err != nil {
		return err
	}
	dest, destOpts, err := getDestWithOptions(context, rule, udfModifier)
	if err != nil {
		return err
	}
	fs, err := StorageService(context, source, dest)
	if err != nil {
		return err
	}
	useCompression := rule.Compress && IsCompressable(source.ParsedURL.Scheme) && IsCompressable(dest.ParsedURL.Scheme)
	object, err := fs.Object(context.Background(), source.URL)
	if err != nil {
		return errors.Wrapf(err, "%v: source not found", source.URL)
	}
	if useCompression {
		err = s.compressSource(context, source, dest, object)
		if err != nil {
			return err
		}
	}
	err = fs.Copy(context.Background(), source.URL, dest.URL, sourceOpts, destOpts)
	if err != nil {
		return err
	}
	if useCompression {
		err = s.decompressTarget(context, source, dest, object)
		if err != nil {
			return err
		}
	}
	response.URLs = append(response.URLs, object.URL())
	return nil
}

// CopyRequest creates a new Copy request
func NewCopyRequest(assets copy.Assets, transfers ...*copy.Rule) *CopyRequest {
	var super *copy.Rule
	if len(transfers) > 0 {
		super = transfers[0]
		transfers = transfers[1:]
	}
	return &CopyRequest{
		Rule:      super,
		Assets:    assets,
		Transfers: transfers,
	}
}

// NewCopyRequestFromURL creates a new request from URL (JSON or YAML format are supported)
func NewCopyRequestFromURL(URL string) (*CopyRequest, error) {
	var request = &CopyRequest{}
	resource := url.NewResource(URL)
	if err := resource.Decode(request); err != nil {
		return nil, err
	}
	return request, nil
}

// Init initialises request
func (r *CopyRequest) Init() error {
	if r.Rule == nil {
		r.Rule = &copy.Rule{}
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
				rule.Source = copy.JoinIfNeeded(r.Source, rule.Source.URL)
			}
			if rule.Dest != nil {
				rule.Dest = copy.JoinIfNeeded(r.Dest, rule.Dest.URL)
			}
		}
		return nil
	}

	if !hasAssets {
		if r.Source != nil && r.Dest != nil {
			r.Transfers = []*copy.Rule{
				r.Rule,
			}
		}
		return nil
	}
	r.Transfers = r.Assets.AsTransfer(r.Rule)
	return nil
}

// Validate checks if request is valid
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
