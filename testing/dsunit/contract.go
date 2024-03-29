package dsunit

import (
	"github.com/viant/assertly"
	"github.com/viant/dsunit"
)

// InitRequest represents an init request
type InitRequest dsunit.InitRequest

// InitResponse represents an init response
type InitResponse dsunit.InitResponse

// RegisterRequest represents a register request
type RegisterRequest dsunit.RegisterRequest

// RegisterResponse represents a register response
type RegisterResponse dsunit.RegisterResponse

// MappingRequest represents a mapping request
type MappingRequest dsunit.MappingRequest

// MappingResponse represents a mapping response
type MappingResponse dsunit.MappingResponse

// type SequenceRequest represents a sequence request
type SequenceRequest dsunit.SequenceRequest

// SequenceResponse represent a sequence response
type SequenceResponse dsunit.SequenceResponse

// RunScriptRequest represents a script request
type RunScriptRequest dsunit.RunScriptRequest

// RunSQLRequest represent run SQL request
type RunSQLRequest dsunit.RunSQLRequest

// RunSQLResponse represents a script response
type RunSQLResponse dsunit.RunSQLResponse

// PrepareRequest represents a prepare request
type PrepareRequest dsunit.PrepareRequest

// PrepareResponse represents a prepare response
type PrepareResponse dsunit.PrepareResponse

// ExpectRequest represents an expect request
type ExpectRequest dsunit.ExpectRequest

// ExpectResponse represent an expect response
type ExpectResponse dsunit.ExpectResponse

// RecreateRequest represents a recreate request
type RecreateRequest dsunit.RecreateRequest

// RecreateResponse represent a recreate response
type RecreateResponse dsunit.RecreateResponse

// QueryRequest represents an query request
type QueryRequest dsunit.QueryRequest

// QueryResponse represents dsunit response
type QueryResponse dsunit.QueryResponse

// FreezeRequest represents a request to create a dataset from existing datastore either for setup or verification
type FreezeRequest dsunit.FreezeRequest

// FreezeResponse represents a freeze response
type FreezeResponse dsunit.FreezeResponse

// DumpRequest represents a request to create a schema DDL from existing database
type DumpRequest dsunit.DumpRequest

// DumpResponse represents a dump response
type DumpResponse dsunit.DumpResponse

// CompareRequest represents a compare request
type CompareRequest dsunit.CompareRequest

// CompareResponse represents a compare response
type CompareResponse dsunit.CompareResponse

// Assertion returns description with validation slice
func (r *ExpectResponse) Assertion() []*assertly.Validation {
	var result = make([]*assertly.Validation, 0)
	if len(r.Validation) == 0 {
		return result
	}
	for _, dbValidation := range r.Validation {
		var validation = dbValidation.Validation
		validation.Description = dbValidation.Dataset
		result = append(result, validation)
	}
	return result
}

// Assertion returns validation slice
func (r *QueryResponse) Assertion() []*assertly.Validation {
	if r == nil || r.Validation == nil {
		return []*assertly.Validation{}
	}
	return []*assertly.Validation{r.Validation}
}

// Assertion returns validation slice
func (r *CompareResponse) Assertion() []*assertly.Validation {
	if r == nil || r.Validation == nil {
		return []*assertly.Validation{}
	}
	return []*assertly.Validation{r.Validation}
}

// CheckSchemaRequest represents a check schema request
type CheckSchemaRequest dsunit.CheckSchemaRequest

// CheckSchemaResponse represents a check schema response
type CheckSchemaResponse dsunit.CheckSchemaResponse

// Assertion returns validation slice
func (r *CheckSchemaResponse) Assertion() []*assertly.Validation {
	var result = make([]*assertly.Validation, 0)
	if r.Validation != nil {
		result = append(result, r.Validation)
	}
	if len(r.Tables) > 0 {
		for i := range r.Tables {
			if validation := r.Tables[i].Validation; validation != nil {
				validation.Description = r.Tables[i].Table
				result = append(result, validation)
			}

		}
	}
	return result
}
