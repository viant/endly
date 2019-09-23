package storage

import (
	"context"
	"github.com/viant/endly"
	"github.com/viant/toolbox/url"
	"github.com/viant/afs"
	"github.com/viant/afs/storage"
	"github.com/viant/afs/scp"
	"github.com/viant/afsc/s3"
	"github.com/viant/afsc/gs"
)

const sshScheme = "ssh"

var fs = afs.New()
var fsFaker = afs.NewFaker()

//GetStorageService return toolbox storage service
func GetStorageService(ctx *endly.Context, resource *url.Resource) (afs.Service, error) {
	var state = ctx.State()
	if state.Has(useMemoryService) {
		return fsFaker, nil
	}
	_ = fs.Close(resource.URL)
	options, err := StorageOpts(ctx, resource)
	if err != nil {
		return nil, err
	}
	return fs, fs.Init(context.Background(), resource.URL, options...)
}

//StorageOpts returns storage option for supplied resource
func StorageOpts(ctx *endly.Context, resource *url.Resource) ([]storage.Option, error) {
	var result = make([]storage.Option, 0)
	if resource.CustomKey != nil {
		if err := resource.CustomKey.Init(); err != nil {
			return nil, err
		}
		if err := resource.CustomKey.Validate(); err != nil {
			return nil, err
		}
		result = append(result, resource.CustomKey)
	}

	if resource.Credentials != "" {
		credConfig, err := ctx.Secrets.GetCredentials(resource.Credentials)
		if err != nil {
			return nil, err
		}
		if credConfig.Region != "" {
			result = append(result, &s3.Region{Name: credConfig.Region})
		}
		payload := ([]byte)(credConfig.Data)
		if resource.ParsedURL != nil {
			switch resource.ParsedURL.Scheme {
			case gs.Scheme:
				auth, err := gs.NewJwtConfig(payload)
				if err != nil {
					return nil, err
				}
				result = append(result, auth)
			case s3.Scheme:
				auth, err := s3.NewAuthConfig(payload)
				if err != nil {
					return nil, err
				}
				result = append(result, auth)
			case scp.Scheme, sshScheme:
				result = append(result, credConfig)
			}
		}
	}
	return result, nil
}
