package storage

import (
	"github.com/viant/afs"
	"github.com/viant/afs/option"
	"github.com/viant/afs/storage"
	"github.com/viant/endly"
	"github.com/viant/endly/system/storage/copy"
	"github.com/viant/toolbox/url"
)

func getSourceWithOptions(context *endly.Context, rule *copy.Rule) (*url.Resource, *option.Source, error) {
	source, err := context.ExpandResource(rule.Source)
	if err != nil {
		return nil, nil, err
	}
	ruleOptions, err := rule.SourceStorageOpts(context)
	if err != nil {
		return nil, nil, err
	}
	sourceOptions, err := StorageOptions(context, source, ruleOptions...)
	if err != nil {
		return nil, nil, err
	}
	return source, option.NewSource(sourceOptions...), nil
}

func getDestWithOptions(context *endly.Context, rule *copy.Rule, modifier option.Modifier) (*url.Resource, *option.Dest, error) {
	ruleOptions, err := rule.DestStorageOpts(context, modifier)
	if err != nil {
		return nil, nil, err
	}
	dest, err := context.ExpandResource(rule.Dest)
	if err != nil {
		return nil, nil, err
	}
	sourceOptions, err := StorageOptions(context, dest, ruleOptions...)
	if err != nil {
		return nil, nil, err
	}
	return dest, option.NewDest(sourceOptions...), nil
}

//GetResourceWithOptions returns resource with afs storage option
func GetResourceWithOptions(context *endly.Context, resource *url.Resource,   options ...storage.Option) (*url.Resource, []storage.Option, error) {
	resource, err := context.ExpandResource(resource)
	if err != nil {
		return nil, nil, err
	}
	sourceOptions, err := StorageOptions(context,  resource)
	if len(options) > 0 {
		sourceOptions = append(sourceOptions, options...)
	}
	return resource, sourceOptions, err
}

//UseMemoryService sets flag on context to always use memory service (testing only)
func UseMemoryService(context *endly.Context) afs.Service {
	state := context.State()
	state.Put(useMemoryService, true)
	return fsFaker
}

//IsCompressable returns true if resource can be compress via shell command.
func IsCompressable(protScheme string) bool {
	return protScheme == "" || protScheme == "scp" || protScheme == "file" || protScheme == "ssh"
}

//Copy transfers data for provided transfer definition.
func Copy(context *endly.Context, transfers ...*copy.Rule) (interface{}, error) {
	if transfers == nil {
		return nil, nil
	}
	transferService, err := context.Service(ServiceID)
	if err != nil {
		return nil, err
	}
	response := transferService.Run(context, &CopyRequest{Transfers: transfers})
	if response.Err != nil {
		return nil, response.Err
	}
	return nil, nil
}
