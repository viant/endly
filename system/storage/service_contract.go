package storage

import (
	"errors"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
)

//Transfer represents copy instruction
type Transfer struct {
	Source   *url.Resource     `required:"true" description:"source asset or directory"`        //source URL with credential
	Target   *url.Resource     `required:"true" description:"destination asset or directory"`   //target URL with credential
	Expand   bool              `description:"flag to substitute asset content with state keys"` //flag to substitute content with state keys
	Compress bool              //flag to compress asset before sending over wirte and to decompress (this option is only supported on scp or file proto)
	Replace  map[string]string `description:"replacements map, if key if found in the conent it wil be replaced with corresponding value."` //replacements map, if key if found in the conent it wil be replaced with corresponding value.
}

//CopyRequest represents a resources copy request
type CopyRequest struct {
	Transfers []*Transfer `required:"true" description:"asset transfers"` // transfers
}

//CopyResponse represents a resources copy response
type CopyResponse struct {
	TransferredURL []string //transferred URLs
}

//DownloadRequest represents a resources download request, it downloads source into context.state target key
type DownloadRequest struct {
	Source    *url.Resource `required:"true" description:"source asset or directory"`
	TargetKey string        `required:"true" description:"state map key destination"`
	Udf       string        `description:"name of udf to transform payload before placing into state map"` //name of udf function that will be used to transform payload
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
	Target    *url.Resource `required:"true" description:"destination asset or directory"` //target URL with credential
}

//UploadResponse represents a upload response
type UploadResponse struct {
	UploadSize int
	UploadURL  string
}

//RemoveRequest represents a resources Remove request
type RemoveRequest struct {
	Resources []*url.Resource `required:"true" description:"resources to remove"`
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
	if r.TargetKey == "" {
		return errors.New("targetKey was empty")
	}
	return nil
}

//Validate checks if request is valid
func (r *RemoveRequest) Validate() error {
	if len(r.Resources) == 0 {
		return errors.New("resources was empty")
	}
	return nil
}

//Validate checks if request is valid
func (t *Transfer) Validate() error {
	if t.Source == nil {
		return errors.New("source was empty")
	}
	if t.Target == nil {
		return errors.New("target was empty")
	}
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
	if r.Target == nil {
		return errors.New("target was empty")
	}
	if r.SourceKey == "" {
		return errors.New("sourceKey was empty")
	}
	return nil
}
