package storage

import (
	"github.com/viant/toolbox/url"
)

type ExistsRequest struct {
	Source *url.Resource `required:"true" description:"source asset or directory"`
	Assets []string `description:"assets paths, joined with source URL"`
	Expect map[string]bool `description:"map of asset and exists flag"`
}
