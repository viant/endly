package storage

import (
	"bytes"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/storage"
	"github.com/viant/toolbox/url"
	"io"
	"io/ioutil"
	"strings"
)

//MaxContentSize represent max allowed expandable content size
var MaxContentSize = 1024 * 64

func canExpand(content []byte) bool {
	if len(content) == 0 {
		return false
	}
	limit := 100
	if limit >= len(content) {
		limit = len(content) - 1
	}
	return toolbox.IsPrintText(string(content[:limit]))
}

//NewExpandedContentHandler return a new reader that can substitute content with state map, replacement data provided in replacement map.
func NewExpandedContentHandler(context *endly.Context, replaceMap map[string]string, expand bool) func(reader io.ReadCloser) (io.ReadCloser, error) {
	return func(reader io.ReadCloser) (io.ReadCloser, error) {
		var replaced = false
		defer reader.Close()
		content, err := ioutil.ReadAll(reader)
		if err != nil {
			return nil, err
		}
		if len(content) > MaxContentSize {
			return ioutil.NopCloser(bytes.NewReader(content)), nil
		}

		var result = string(content)
		if expand && canExpand(content) {
			result = context.Expand(result)
			if err != nil {
				return nil, err
			}
			replaced = len(result) != len(content)
		}

		for k, v := range replaceMap {
			if !replaced && strings.Contains(result, k) {
				replaced = true
			}
			result = strings.Replace(result, k, v, len(result))
		}
		if replaced {
			return ioutil.NopCloser(strings.NewReader(toolbox.AsString(result))), nil
		}
		return ioutil.NopCloser(bytes.NewReader(content)), nil
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
	if resource.Credentials != "" {
		var err error
		if resource.Credentials, err = context.Secrets.CredentialsLocation(resource.Credentials); err != nil {
			return nil, err
		}
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
	} else if !(strings.HasPrefix(URI, "/") || strings.HasPrefix(URI, "$")) {
		var hostname = parent.ParsedURL.Hostname()
		if hostname == "" || hostname == "127.0.0.1" || hostname == "localhost" {
			var candidate = url.NewResource(URI)
			if toolbox.FileExists(candidate.ParsedURL.Path) {
				result = candidate
			}
		}
		if result == nil {
			result = url.NewResource(toolbox.URLPathJoin(parent.URL, URI))
		}

	} else if parent != nil {
		result = url.NewResource(toolbox.URLPathJoin(parent.URL, URI))
	} else {
		result = url.NewResource(URI)
	}
	if strings.HasPrefix(URI, "$") { //has to expand to be re-evaluated
		result.URL = URI
	}
	return result
}
