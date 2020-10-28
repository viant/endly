package http

import (
	"fmt"
	"github.com/viant/toolbox/bridge"
	"net/http"
	"strings"
)

//HeaderProvider return a header value for supplied source
func HeaderProvider(header string) HTTPRequestKeyProvider {
	return func(source interface{}) (string, error) {
		switch request := source.(type) {
		case *bridge.HttpRequest:
			return strings.Join(request.Header[header], "\n"), nil
		case *http.Request:
			return strings.Join(request.Header[header], "\n"), nil
		}
		return "", fmt.Errorf("unsupported request type %T", source)
	}
}

