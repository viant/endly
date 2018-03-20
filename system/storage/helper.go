package storage

import (
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/storage"
	"github.com/viant/toolbox/url"
	"io"
	"io/ioutil"
	"strings"
)

//NewExpandedContentHandler return a new reader that can substitute content with state map, replacement data provided in replacement map.
func NewExpandedContentHandler(context *endly.Context, replaceMap map[string]string, expand bool) func(reader io.ReadCloser) (io.ReadCloser, error) {
	return func(reader io.ReadCloser) (io.ReadCloser, error) {
		defer reader.Close()
		content, err := ioutil.ReadAll(reader)
		if err != nil {
			return nil, err
		}
		var result = string(content)
		if expand {
			result = context.Expand(result)
			if err != nil {
				return nil, err
			}
		}
		for k, v := range replaceMap {
			result = strings.Replace(result, k, v, len(result))
		}
		return ioutil.NopCloser(strings.NewReader(toolbox.AsString(result))), nil
	}
}

//UseMemoryService sets flag on context to always use memory service (testing only)
func UseMemoryService(context *endly.Context) storage.Service {
	state := context.State()
	state.Put(useMemoryService, true)
	return storage.NewMemoryService()
}

//GetStorageService return toolbox storage service
func GetStorageService(context *endly.Context, resource *url.Resource) (storage.Service, error) {
	var state = context.State()
	if state.Has(useMemoryService) {
		return storage.NewMemoryService(), nil
	}
	return storage.NewServiceForURL(resource.URL, resource.Credentials)
}

//IsShellCompressable returns true if resource can be compress via shell command.
func IsShellCompressable(protScheme string) bool {
	return protScheme == "scp" || protScheme == "file"
}

//Copy transfers data for provided transfer definition.
func Copy(context *endly.Context, transfers ...*Transfer) (interface{}, error) {
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

func joinIfNeeded(parent *url.Resource, URI string) (result *url.Resource) {
	defer func() {
		if parent != nil {
			result.Credentials = parent.Credentials
		}
	}()
	if strings.Contains(URI, ":/") {
		result = url.NewResource(URI)
	} else if parent != nil {
		result = url.NewResource(toolbox.URLPathJoin(parent.URL, URI))
	} else {
		result = url.NewResource(URI)
	}
	return result
}
