package dbunit

import (
	"github.com/viant/assertly"
	core "github.com/viant/dbunit"
	"github.com/viant/dbunit/fixture"
)

// Assets selects a session-bound resolver or an explicit absolute fixture root.
type Assets struct {
	Fixtures string `json:"fixtures,omitempty" description:"Session fixture resolver name, including embedded assets"`
	Root     string `json:"root,omitempty" description:"Explicit absolute fixture directory; mutually exclusive with fixtures"`
	Capture  string `json:"capture,omitempty" description:"Output state path; defaults to dbunit.<database>"`
}
type RegisterRequest struct {
	Database    string                  `json:"database" required:"true" description:"Session-local connection name"`
	Driver      string                  `json:"driver" required:"true" description:"database/sql driver; sqlite uses the pure Go driver"`
	DSN         string                  `json:"dsn" required:"true" description:"Driver connection string; credential placeholders are {username} and {password}"`
	Credentials string                  `json:"credentials,omitempty" description:"Credential registry alias or secret URL"`
	Dialect     string                  `json:"dialect,omitempty" description:"Explicit SQL dialect when it cannot be detected"`
	Replace     bool                    `json:"replace,omitempty" description:"Replace connection after outstanding operations finish"`
	NativeLoad  *core.NativeLoadOptions `json:"nativeLoad,omitempty" description:"Explicit caller certification for the selected SQLX native loader; disabled by default"`
}
type RegisterResponse struct {
	Database string `json:"database"`
	Driver   string `json:"driver"`
}
type PrepareRequest struct {
	DatasetsFrom    string               `json:"datasetsFrom,omitempty" description:"Explicit workflow state path to hydrated/mapped datasets; mutually exclusive with datasets"`
	Database        string               `json:"database" required:"true"`
	TransactionMode core.TransactionMode `json:"transactionMode,omitempty" description:"prepareAtomic (default) or nonAtomic; caller-owned transaction modes require Go APIs"`
	Assets
	core.PrepareRequest
}
type ExpectRequest struct {
	Database string `json:"database" required:"true"`
	Assets
	core.ExpectRequest
}
type QueryRequest struct {
	Assets
	core.QueryRequest
}
type ScriptRequest struct {
	Assets
	core.SchemaApplyRequest
}
type InitRequest struct {
	RegisterRequest
	Assets
	Sources    []fixture.Source `json:"sources,omitempty"`
	Statements []string         `json:"statements,omitempty"`
}
type SQLRequest struct {
	core.ExecRequest
	TransactionMode core.TransactionMode `json:"transactionMode,omitempty" description:"prepareAtomic (default) or nonAtomic; Go-owned transaction modes are not serializable"`
	Capture         string               `json:"capture,omitempty"`
}
type SQLResponse = core.ExecResult
type DumpRequest struct {
	core.SchemaExtractRequest
	Destination string `json:"destination,omitempty" description:"Explicit absolute writable SQL destination"`
	Capture     string `json:"capture,omitempty"`
}
type ExpectResponse struct{ *core.Report }

// Assertion adapts the completed report. It never reevaluates predicates or queries.
func (r *ExpectResponse) Assertion() []*assertly.Validation {
	if r == nil || r.Report == nil {
		return nil
	}
	validation := &assertly.Validation{Description: r.Name, PassedCount: r.Passed, FailedCount: r.Failed}
	for _, mismatch := range r.Mismatches {
		validation.Failures = append(validation.Failures, &assertly.Failure{Source: mismatch.Expectation, Path: mismatch.Path, Expected: mismatch.Expected, Actual: mismatch.Actual, Reason: mismatch.Category, Message: mismatch.Message})
	}
	return []*assertly.Validation{validation}
}

// ExportRequest exports a bounded query to explicit JSON/NDJSON content.
type ExportRequest struct {
	Database string `json:"database" required:"true"`
	core.ExportRequest
	Destination string `json:"destination,omitempty" description:"Explicit absolute writable fixture destination"`
	Capture     string `json:"capture,omitempty"`
}

// CompareRequest compares explicit queries using registered session connections.
type CompareRequest struct {
	Database string `json:"database" required:"true"`
	core.CompareRequest
	Capture string `json:"capture,omitempty"`
}

// HydrateRequest generates concrete rows and explicit captures using core dbunit.
type HydrateRequest struct {
	Database string `json:"database" required:"true"`
	Assets
	core.HydrateRequest
}
type SequenceRequest struct {
	Database string `json:"database" required:"true"`
	Capture  string `json:"capture,omitempty"`
	core.SequenceRequest
}
type MapRequest struct {
	SourceFrom string `json:"sourceFrom,omitempty" description:"Explicit workflow state path to a resolved fixture source; mutually exclusive with source"`
	Database   string `json:"database" required:"true"`
	Assets
	core.MapRequest
}
