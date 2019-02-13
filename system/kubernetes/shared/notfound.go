package shared

import "strings"

//NotFound represents not found error
type NotFound struct {
	Message string
}

func (f *NotFound) Error() string {
	return f.Message
}

//IsNotFound returns true if error is not found
func IsNotFound(candidate interface{}) bool {
	if candidate == nil {
		return false
	}
	switch candidate.(type) {
	case NotFound:
		return true
	case *NotFound:
		return true
	}
	err, ok := candidate.(error)
	if !ok {
		return false
	}
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "not found") || strings.Contains(strings.ToLower(err.Error()), "not find")
}
