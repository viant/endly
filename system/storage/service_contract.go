package storage

import (
	"errors"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
)


//AssetTransfer represents asset transfer
type AssetTransfer map[string]interface{}

//Transfer represents copy instruction
type Transfer struct {
	Expand   bool              `description:"flag to substitute asset content with state keys"`
	Compress bool              `description:"flag to compress asset before sending over wire and to decompress (this option is only supported on scp or file scheme)"` //flag to compress asset before sending over wirte and to decompress (this option is only supported on scp or file proto)
	Replace  map[string]string `description:"replacements map, if key if found in the conent it wil be replaced with corresponding value."`
	Source   *url.Resource     `required:"true" description:"source asset or directory"`
	Dest     *url.Resource     `required:"true" description:"destination asset or directory"`
}

//NewTransfer creates a new transfer
func NewTransfer(source, dest *url.Resource, compress, expand bool, replace map[string]string) *Transfer {
	return &Transfer{
		Source:   source,
		Dest:     dest,
		Compress: compress,
		Expand:   expand,
		Replace:  replace,
	}
}

//CopyRequest represents a resources copy request
type CopyRequest struct {
	*Transfer               ` description:"if asset uses relative path it will be joined with this URL"`
	Assets    AssetTransfer `description:"map entry can either represent a transfer struct or simple key is the source and the value destination relative path"` // transfers
	Transfers []*Transfer   `description:"actual transfer assets, if empty it derives from assets or source/desc "`
}

//CopyRequest creates a new copy request
func NewCopyRequest(assets AssetTransfer, transfers ...*Transfer) *CopyRequest {
	var super *Transfer
	if len(transfers) > 0 {
		super = transfers[0]
		transfers = transfers[1:]
	}
	return &CopyRequest{
		Transfer:  super,
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

//CopyResponse represents a resources copy response
type CopyResponse struct {
	TransferredURL []string //transferred URLs
}

//DownloadRequest represents a resources download request, it downloads source into context.state target key
type DownloadRequest struct {
	Source  *url.Resource `required:"true" description:"source asset or directory"`
	DestKey string        `required:"true" description:"state map key destination"`
	Udf     string        `description:"name of udf to transform payload before placing into state map"` //name of udf function that will be used to transform payload
}

//DownloadResponse represents a download response
type DownloadResponse struct {
	Info        toolbox.FileInfo
	Payload     string //source content, if binary then is will be prefixed base64: followed by based 64 encoded content.
	Transformed interface{}
}

//UploadRequest represents a resources upload request, it takes context state key to upload to target destination.
type UploadRequest struct {
	SourceKey string        `required:"true" description:"state key with asset content"`
	Dest      *url.Resource `required:"true" description:"destination asset or directory"` //target URL with credentials
}

//UploadResponse represents a upload response
type UploadResponse struct {
	UploadSize int
	UploadURL  string
}

//RemoveRequest represents a resources Remove request
type RemoveRequest struct {
	Assets []*url.Resource `required:"true" description:"resources to remove"`
}

//NewRemoveRequest creates a new remove request
func NewRemoveRequest(assets ...*url.Resource) *RemoveRequest {
	return &RemoveRequest{
		Assets: assets,
	}
}

//RemoveResponse represents a resources Remove response, it returns url of all resource that have been removed.
type RemoveResponse struct {
	Removed []string
}

//Validate checks if request is valid
func (r *DownloadRequest) Validate() error {
	if r.Source == nil {
		return errors.New("source was empty")
	}
	if r.DestKey == "" {
		return errors.New("targetKey was empty")
	}
	return nil
}

//Validate checks if request is valid
func (r *RemoveRequest) Validate() error {
	if len(r.Assets) == 0 {
		return errors.New("assets was empty")
	}
	return nil
}

//Validate checks if request is valid
func (t *Transfer) Validate() error {
	if t.Source == nil {
		return errors.New("source was empty")
	}
	if t.Source.URL == "" {
		return errors.New("source.URL was empty")
	}
	if t.Dest == nil {
		return errors.New("dest was empty")
	}
	if t.Dest.URL == "" {
		return errors.New("dest.URL was empty")
	}
	return nil
}

//Init initialises request
func (r *CopyRequest) Init() error {
	if r.Transfer == nil {
		r.Transfer = &Transfer{}
	}

	hasAssets := len(r.Assets) > 0
	hasTransfers := len(r.Transfers) > 0
	if hasTransfers {
		if r.Source == nil && r.Dest == nil {
			return nil
		}
		for _, transfer := range r.Transfers {
			if transfer.Source != nil {
				transfer.Source = joinIfNeeded(r.Source, transfer.Source.URL)
			}
			if transfer.Dest != nil {
				transfer.Dest = joinIfNeeded(r.Dest, transfer.Dest.URL)
			}
		}
		return nil
	}

	if !hasAssets {
		if r.Source != nil && r.Dest != nil {
			r.Transfers = []*Transfer{
				{
					Source: r.Source,
					Dest:   r.Dest,
				},
			}
		}
		return nil
	}
	r.Transfers = r.Assets.AsTransfer(r.Source, r.Dest)
	return nil
}

//Validate checks if request is valid
func (r *CopyRequest) Validate() error {
	if len(r.Transfers) == 0 {
		return errors.New("transfers were empty")
	}
	for _, transfer := range r.Transfers {
		if err := transfer.Validate(); err != nil {
			return err
		}
	}
	return nil
}

//Validate checks if request is valid
func (r *UploadRequest) Validate() error {
	if r.Dest == nil {
		return errors.New("dest was empty")
	}
	if r.SourceKey == "" {
		return errors.New("sourceKey was empty")
	}
	return nil
}

//AsTransfer converts map to transfer or transfers
func (t *AssetTransfer) AsTransfer(sourceBase, destBase *url.Resource) []*Transfer {
	aMap := toolbox.AsMap(t)
	var transfers = make([]*Transfer, 0)
	var isSourceRootPath = sourceBase != nil && sourceBase.ParsedURL != nil && sourceBase.ParsedURL.Path == "/"
	var isDestRootPath = destBase != nil && destBase.ParsedURL != nil && destBase.ParsedURL.Path == "/"
	for source, v := range aMap {
		if v == nil {
			v = source
		}
		var dest, ok = v.(string)
		if !ok {
			continue
		}
		if isSourceRootPath {
			source = url.NewResource(source).URL
		}
		if isDestRootPath {
			dest = url.NewResource(dest).URL
		}
		transfers = append(transfers, &Transfer{
			Source: joinIfNeeded(sourceBase, source),
			Dest:   joinIfNeeded(destBase, dest),
		})
	}
	return transfers
}
