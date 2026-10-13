package http

import (
	"fmt"
	"github.com/viant/toolbox/bridge"
	"net/http"
	"strings"
)

// HeaderProvider return a header value for supplied source
func HeaderProvider(header string) HTTPRequestKeyProvider {
	return func(source interface{}) (string, error) {
		switch request := source.(type) {
		case *bridge.HttpRequest:
			for name, values := range request.Header {
				if strings.EqualFold(name, header) {
					return strings.Join(values, "\n"), nil
				}
			}
			return "", nil
		case *http.Request:
			for name, values := range request.Header {
				if strings.EqualFold(name, header) {
					return strings.Join(values, "\n"), nil
				}
			}
			return "", nil
		}
		return "", fmt.Errorf("unsupported request type %T", source)
	}
}
