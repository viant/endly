package http

import (
	"fmt"
	"github.com/viant/endly/internal/util"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/bridge"
	"io/ioutil"
	"net/http"
	"strings"
)

func initProviders() {
	HTTPRequestKeyProviders[URLKey] = func(source interface{}) (string, error) {
		switch request := source.(type) {
		case *bridge.HttpRequest:

			return stripProtoAndHost(request.URL), nil
		case *http.Request:
			return stripProtoAndHost(request.URL.String()), nil
		}
		return "", fmt.Errorf("unsupported request type %T", source)
	}
	HTTPRequestKeyProviders[MethodKey] = func(source interface{}) (string, error) {
		switch request := source.(type) {
		case *bridge.HttpRequest:
			return request.Method, nil
		case *http.Request:
			return request.Method, nil
		}
		return "", fmt.Errorf("unsupported request type %T", source)
	}
	HTTPRequestKeyProviders[CookieKey] = HeaderProvider(CookieKey)
	HTTPRequestKeyProviders[ContentTypeKey] = HeaderProvider(ContentTypeKey)
	HTTPRequestKeyProviders[BodyKey] = func(source interface{}) (string, error) {

		switch request := source.(type) {
		case *bridge.HttpRequest:
			body, _ := util.FromPayload(request.Body)
			return util.AsPayload(body), nil
		case *http.Request:
			if request.ContentLength == 0 {
				return "", nil
			}
			content, err := ioutil.ReadAll(request.Body)
			if err != nil {
				return "", fmt.Errorf("failed to read body %v, %v", request.URL, err)
			}
			encoded := string(content)
			if strings.HasPrefix(encoded, "base64:") {
				content, _ = util.FromPayload(encoded)
			}
			return util.AsPayload(content), nil
		}
		return "", fmt.Errorf("unsupported request type %T", source)
	}
}

func stripProtoAndHost(URL string) string {
	if index := strings.Index(URL, "://"); index != -1 {
		URL = string(URL[index+3:])
	}
	if index := strings.Index(URL, "/"); index > 0 {
		URL = string(URL[index:])
	}
	return URL
}

func buildKeyValue(keys []string, request interface{}) (string, error) {
	var values = make([]string, 0)
	for _, key := range keys {
		provider, has := HTTPRequestKeyProviders[key]
		if !has {
			if strings.HasPrefix(key, "Header.") {
				HTTPRequestKeyProviders[key] = HeaderProvider(strings.Replace(key, "Header.", "", 1))
				provider = HTTPRequestKeyProviders[key]
			} else {
				return "", fmt.Errorf("unsupported key: %v, available, [%v]", key, strings.Join(toolbox.MapKeysToStringSlice(HTTPRequestKeyProviders), ","))
			}
		}
		value, err := provider(request)
		if err != nil {
			return "", fmt.Errorf("unable to get value for %v, %v", key, err)
		}
		values = append(values, value)
	}
	return strings.Join(values, ","), nil
}
