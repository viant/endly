package endly

import (
	"github.com/viant/dsunit"
	"github.com/viant/toolbox/url"
)

//DsUnitMappingRequest represents a mapping request
type DsUnitMappingRequest struct {
	Mappings []*url.Resource //Resource pointing to JSON representation of *dsunit.DatasetMapping
}

//DsUnitMappingResponse represents mapping response
type DsUnitMappingResponse struct {
	Tables []string //all individual tables that are defined as a mapping
}

//DatasetMapping represnts dataset mapping
type DatasetMapping struct {
	Name  string                 //mapping Id
	Value *dsunit.DatasetMapping //actual mappings
}
