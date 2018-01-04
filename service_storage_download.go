package endly

import (
	"errors"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/url"
)

//StorageDownloadRequest represents a resources download request, it downloads source into context.state target key
type StorageDownloadRequest struct {
	Source    *url.Resource
	TargetKey string
}

//Validate checks if request is valid
func (r *StorageDownloadRequest) Validate() error {
	if r.Source == nil {
		return errors.New("source was empty")
	}
	if r.TargetKey == "" {
		return errors.New("targetKey was empty")
	}
	return nil
}

//StorageDownloadResponse represents a download response
type StorageDownloadResponse struct {
	Info    toolbox.FileInfo
	Payload string
}
