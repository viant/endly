package udf

import (
	"github.com/viant/endly"
	"github.com/viant/endly/model/location"
)

// RegisterRequest represents a register udf request
type RegisterRequest struct {
	UDFs []*endly.UdfProvider `description:"collection of predefined udf provider name with custom parameters and new registration id"`
}

func NewRegisterRequestFromURL(URL string) (*RegisterRequest, error) {
	var request = &RegisterRequest{}
	resource := location.NewResource(URL)
	return request, resource.Decode(request)
}

// RegisterRequest represents a register response
type RegisterResponse struct{}
