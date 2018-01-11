package endly

import (
	"errors"
	"github.com/viant/toolbox/url"
)

//StorageUploadRequest represents a resources upload request, it takes context state key to upload to target destination.
type StorageUploadRequest struct {
	SourceKey string
	Target    *url.Resource
}

//Validate checks if request is valid
func (r *StorageUploadRequest) Validate() error {
	if r.Target == nil {
		return errors.New("Target was empty")
	}
	if r.SourceKey == "" {
		return errors.New("SourceKey was empty")
	}
	return nil
}

//StorageUploadResponse represents a upload response
type StorageUploadResponse struct {
	UploadSize int
	UploadURL  string
}
