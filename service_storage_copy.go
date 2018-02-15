package endly

import (
	"github.com/viant/toolbox/url"
	"github.com/pkg/errors"
)

//StorageCopyRequest represents a resources copy request
type StorageCopyRequest struct {
	Transfers []*Transfer `required:"true" description:"asset transfers"` // transfers
}

//StorageCopyResponse represents a resources copy response
type StorageCopyResponse struct {
	TransferredURL []string //transferred URLs
}

//Transfer represents copy instruction
type Transfer struct {
	Source   *url.Resource     `required:"true" description:"source asset or directory"`                                                    //source URL with credential
	Target   *url.Resource     `required:"true" description:"destination asset or directory"`                                               //target URL with credential
	Expand   bool              `description:"flag to substitute asset content with state keys"`                                             //flag to substitute content with state keys
	Compress bool                                                                                                                           //flag to compress asset before sending over wirte and to decompress (this option is only supported on scp or file proto)
	Replace  map[string]string `description:"replacements map, if key if found in the conent it wil be replaced with corresponding value."` //replacements map, if key if found in the conent it wil be replaced with corresponding value.
}

func (t *Transfer) Validate() error {
	if t.Source == nil {
		return errors.New("source was empty")
	}
	if t.Target == nil {
		return errors.New("target was empty")
	}
	return nil
}

func (r *StorageCopyRequest) Validate() error {
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
