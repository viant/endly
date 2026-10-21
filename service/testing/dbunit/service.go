package dbunit

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	core "github.com/viant/dbunit"
	"github.com/viant/dbunit/fixture"
	"github.com/viant/endly"
	_ "github.com/viant/sqlx/metadata/product/sqlite"
	"github.com/viant/toolbox/data"
	_ "modernc.org/sqlite"
)

const ServiceID = "dbunit"

type service struct{ *endly.AbstractService }

func New() endly.Service {
	s := &service{AbstractService: endly.NewAbstractService(ServiceID)}
	s.route("register", func() any { return &RegisterRequest{} }, func() any { return &RegisterResponse{} }, func(ctx *endly.Context, r any) (any, error) { return s.register(ctx, r.(*RegisterRequest)) })
	s.route("init", func() any { return &InitRequest{} }, func() any { return &core.SchemaResult{} }, func(ctx *endly.Context, r any) (any, error) {
		req := r.(*InitRequest)
		if _, err := s.register(ctx, &req.RegisterRequest); err != nil {
			return nil, err
		}
		return s.script(ctx, &ScriptRequest{Assets: req.Assets, SchemaApplyRequest: core.SchemaApplyRequest{Database: req.Database, Sources: req.Sources, Statements: req.Statements}})
	})
	s.route("script", func() any { return &ScriptRequest{} }, func() any { return &core.SchemaResult{} }, func(ctx *endly.Context, r any) (any, error) { return s.script(ctx, r.(*ScriptRequest)) })
	s.route("sql", func() any { return &SQLRequest{} }, func() any { return &SQLResponse{} }, func(ctx *endly.Context, r any) (any, error) { return s.sql(ctx, r.(*SQLRequest)) }, "statements", "state")
	s.route("prepare", func() any { return &PrepareRequest{} }, func() any { return &core.PrepareResult{} }, func(ctx *endly.Context, r any) (any, error) { return s.prepare(ctx, r.(*PrepareRequest)) }, "datasets", "state")
	s.route("expect", func() any { return &ExpectRequest{} }, func() any { return &ExpectResponse{} }, func(ctx *endly.Context, r any) (any, error) { return s.expect(ctx, r.(*ExpectRequest)) }, "expectations", "state")
	s.route("query", func() any { return &QueryRequest{} }, func() any { return &core.QueryResult{} }, func(ctx *endly.Context, r any) (any, error) { return s.query(ctx, r.(*QueryRequest)) }, "args", "state")
	s.route("dump", func() any { return &DumpRequest{} }, func() any { return &core.SchemaArtifact{} }, func(ctx *endly.Context, r any) (any, error) { return s.dump(ctx, r.(*DumpRequest)) })
	s.route("export", func() any { return &ExportRequest{} }, func() any { return &core.ExportResult{} }, func(ctx *endly.Context, r any) (any, error) { return s.export(ctx, r.(*ExportRequest)) }, "query")
	s.route("compare", func() any { return &CompareRequest{} }, func() any { return &ExpectResponse{} }, func(ctx *endly.Context, r any) (any, error) { return s.compare(ctx, r.(*CompareRequest)) }, "expected", "actual")
	s.route("hydrate", func() any { return &HydrateRequest{} }, func() any { return &core.HydrateResult{} }, func(ctx *endly.Context, r any) (any, error) { return s.hydrate(ctx, r.(*HydrateRequest)) }, "tables", "sequences", "state", "stateCaptures")
	s.route("sequence", func() any { return &SequenceRequest{} }, func() any { return &core.SequenceResult{} }, func(ctx *endly.Context, r any) (any, error) { return s.sequence(ctx, r.(*SequenceRequest)) }, "sequences", "state")
	s.route("map", func() any { return &MapRequest{} }, func() any { return &core.MapResult{} }, func(ctx *endly.Context, r any) (any, error) { return s.mapFixture(ctx, r.(*MapRequest)) }, "mapping", "source", "state")
	mappingRoute, _ := s.Route("map")
	mappingRoute.OnRawRequest = decodeMappingRequest
	return s
}
func (s *service) route(action string, request, response func() any, handler func(*endly.Context, any) (any, error), deferred ...string) {
	s.Register(&endly.Route{Action: action, RequestInfo: &endly.ActionInfo{Description: "SQL fixture " + action + " using session-local database/sql connections"}, RequestProvider: request, ResponseProvider: response, Handler: handler, DeferredRequestFields: deferred, ExpandRequest: true})
}

func (s *service) register(ctx *endly.Context, req *RegisterRequest) (*RegisterResponse, error) {
	if req.Database == "" || req.Driver == "" || req.DSN == "" {
		return nil, fmt.Errorf("database, driver and dsn are required")
	}
	holder, err := sessionFor(ctx)
	if err != nil {
		return nil, err
	}
	// Waiting for existing readers also gives replacement a well-defined lifetime.
	holder.mu.Lock()
	defer holder.mu.Unlock()
	if holder.closed || holder.done.Err() != nil {
		return nil, fmt.Errorf("dbunit session is closed")
	}
	previous := holder.databases[req.Database]
	if previous != nil && !req.Replace {
		return nil, fmt.Errorf("database %q is already registered", req.Database)
	}
	op, cancel := context.WithCancel(ctx.Background())
	defer cancel()
	stop := context.AfterFunc(holder.done, cancel)
	defer stop()
	driver := req.Driver
	switch driver {
	case "sqlite3", "moderncsqlite":
		driver = "sqlite"
	case "postgresql":
		driver = "postgres"
	}
	dsn := req.DSN
	if req.Credentials != "" {
		if ctx.Secrets == nil {
			return nil, fmt.Errorf("credential lookup is unavailable")
		}
		credentials, err := ctx.Secrets.GetCredentials(op, req.Credentials)
		if err != nil {
			return nil, fmt.Errorf("resolve database credentials: %w", err)
		}
		dsn = strings.NewReplacer("{username}", credentials.Username, "{password}", credentials.Password).Replace(dsn)
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("open database %q with driver %q: %w", req.Database, driver, err)
	}
	if err = db.PingContext(op); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping database %q: %w", req.Database, err)
	}
	dialect := req.Dialect
	if dialect == "" {
		dialect = driver
	}
	if _, err = core.New(db, core.WithDialect(dialect)); err != nil {
		_ = db.Close()
		return nil, err
	}
	binding := &connection{db: db, owned: true, driver: dialect}
	if req.NativeLoad != nil {
		capability := *req.NativeLoad
		binding.nativeLoad = &capability
	}
	holder.databases[req.Database] = binding
	if previous != nil && previous.owned {
		_ = previous.db.Close()
	}
	return &RegisterResponse{Database: req.Database, Driver: driver}, nil
}

func engine(ctx *endly.Context, database string, assets Assets) (*core.Engine, context.Context, func(), error) {
	holder, err := sessionFor(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	op, done, err := holder.operation(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	fail := func(err error) (*core.Engine, context.Context, func(), error) { done(); return nil, nil, nil, err }
	binding := holder.databases[database]
	if binding == nil {
		return fail(fmt.Errorf("unregistered database %q", database))
	}
	options := []core.Option{}
	if binding.nativeLoad != nil {
		options = append(options, core.WithNativeLoad(*binding.nativeLoad))
	}
	if binding.driver != "" {
		options = append(options, core.WithDialect(binding.driver))
	}
	for name, item := range holder.databases {
		options = append(options, core.WithDatabase(name, item.db))
	}
	if assets.Fixtures != "" && assets.Root != "" {
		return fail(fmt.Errorf("fixtures and root are mutually exclusive"))
	}
	if assets.Fixtures != "" {
		resolver := holder.fixtures[assets.Fixtures]
		if resolver == nil {
			return fail(fmt.Errorf("unregistered fixtures %q", assets.Fixtures))
		}
		options = append(options, core.WithFixtures(resolver))
	}
	if assets.Root != "" {
		if !filepath.IsAbs(assets.Root) {
			return fail(fmt.Errorf("fixture root must be absolute"))
		}
		options = append(options, core.WithFixtures(fixture.Directory(assets.Root)))
	}
	result, err := core.New(binding.db, options...)
	if err != nil {
		return fail(err)
	}
	return result, op, done, nil
}

func (s *service) script(ctx *endly.Context, req *ScriptRequest) (*core.SchemaResult, error) {
	e, op, done, err := engine(ctx, req.Database, req.Assets)
	if err != nil {
		return nil, err
	}
	defer done()
	result, err := e.ApplySchema(op, req.SchemaApplyRequest)
	if err == nil {
		capture(ctx, req.Capture, req.Database, result)
	}
	return result, err
}
func (s *service) prepare(ctx *endly.Context, req *PrepareRequest) (*core.PrepareResult, error) {
	mode, err := workflowTransactionMode(req.TransactionMode)
	if err != nil {
		return nil, err
	}
	e, op, done, err := engine(ctx, req.Database, req.Assets)
	if err != nil {
		return nil, err
	}
	defer done()
	spec := req.PrepareRequest
	spec.State = snapshot(ctx, spec.State)
	if req.DatasetsFrom != "" {
		if len(spec.Datasets) > 0 {
			return nil, fmt.Errorf("datasetsFrom and datasets are mutually exclusive")
		}
		spec.Datasets, err = datasetsFrom(spec.State, req.DatasetsFrom)
		if err != nil {
			return nil, err
		}
	}
	spec.Datasets = append([]core.Dataset(nil), spec.Datasets...)
	for i := range spec.Datasets {
		item := &spec.Datasets[i]
		item.Table = expandTable(spec.State, item.Table)
		item.Database = spec.State.ExpandAsText(item.Database)
		if item.Database != "" && item.Database != req.Database {
			return nil, fmt.Errorf("prepare dataset must use selected database %q", req.Database)
		}
		item.Database = ""
		item.Source.Path = spec.State.ExpandAsText(item.Source.Path)
		item.Source.URL = spec.State.ExpandAsText(item.Source.URL)
	}
	result, err := e.Prepare(op, spec, core.WithTransactionMode(mode))
	if err == nil {
		capture(ctx, req.Capture, req.Database, result)
	}
	return result, err
}
func expandTable(state data.Map, table core.Identifier) core.Identifier {
	result := append(core.Identifier(nil), table...)
	for i := range result {
		result[i] = state.ExpandAsText(result[i])
	}
	return result
}
func (s *service) expect(ctx *endly.Context, req *ExpectRequest) (*ExpectResponse, error) {
	e, op, done, err := engine(ctx, req.Database, req.Assets)
	if err != nil {
		return nil, err
	}
	defer done()
	spec := req.ExpectRequest
	spec.State = snapshot(ctx, spec.State)
	spec.Expectations = append([]core.Expectation(nil), spec.Expectations...)
	for i := range spec.Expectations {
		item := &spec.Expectations[i]
		item.Table = expandTable(spec.State, item.Table)
		item.Database = spec.State.ExpandAsText(item.Database)
		if len(item.Table) > 0 {
			if item.Database != "" && item.Database != req.Database {
				return nil, fmt.Errorf("table expectation must use selected database %q", req.Database)
			}
			item.Database = ""
		} else if item.Database == req.Database {
			item.Database = ""
		}
		item.Expected.Path = spec.State.ExpandAsText(item.Expected.Path)
		item.Expected.URL = spec.State.ExpandAsText(item.Expected.URL)
		if item.Query != nil {
			query := *item.Query
			query.SQL = spec.State.ExpandAsText(query.SQL)

			item.Query = &query
		}
	}
	report, err := e.Expect(op, spec)
	result := &ExpectResponse{Report: report}
	if err == nil {
		capture(ctx, req.Capture, req.Database, report)
	}
	return result, err
}
func (s *service) query(ctx *endly.Context, req *QueryRequest) (*core.QueryResult, error) {
	e, op, done, err := engine(ctx, req.Database, req.Assets)
	if err != nil {
		return nil, err
	}
	defer done()
	spec := req.QueryRequest
	spec.State = snapshot(ctx, spec.State)
	result, err := e.Query(op, spec)
	if err == nil {
		capture(ctx, req.Capture, req.Database, result)
	}
	return result, err
}
func (s *service) sql(ctx *endly.Context, req *SQLRequest) (*SQLResponse, error) {
	mode, err := workflowTransactionMode(req.TransactionMode)
	if err != nil {
		return nil, err
	}
	e, op, done, err := engine(ctx, req.Database, Assets{})
	if err != nil {
		return nil, err
	}
	defer done()
	spec := req.ExecRequest
	spec.State = snapshot(ctx, spec.State)
	result, err := e.Exec(op, spec, core.WithTransactionMode(mode))
	if err == nil {
		capture(ctx, req.Capture, req.Database, result)
	}
	return result, err
}
func workflowTransactionMode(mode core.TransactionMode) (core.TransactionMode, error) {
	switch mode {
	case "", core.PrepareAtomic:
		return core.PrepareAtomic, nil
	case core.ExplicitNonAtomic:
		return core.ExplicitNonAtomic, nil
	default:
		return "", fmt.Errorf("transactionMode %q requires the Go transaction APIs; workflow actions accept prepareAtomic or nonAtomic", mode)
	}
}
func (s *service) dump(ctx *endly.Context, req *DumpRequest) (*core.SchemaArtifact, error) {
	if req.Destination != "" && !filepath.IsAbs(req.Destination) {
		return nil, fmt.Errorf("dump destination must be absolute")
	}
	e, op, done, err := engine(ctx, req.Database, Assets{})
	if err != nil {
		return nil, err
	}
	defer done()
	result, err := e.ExtractSchema(op, req.SchemaExtractRequest)
	if err != nil {
		return result, err
	}
	if req.Destination != "" {
		if err = os.WriteFile(req.Destination, []byte(result.SQL), 0600); err != nil {
			return result, err
		}
	}
	capture(ctx, req.Capture, req.Database, result)
	return result, nil
}

func (s *service) export(ctx *endly.Context, req *ExportRequest) (*core.ExportResult, error) {
	if req.Destination != "" && !filepath.IsAbs(req.Destination) {
		return nil, fmt.Errorf("export destination must be absolute")
	}
	e, op, done, err := engine(ctx, req.Database, Assets{})
	if err != nil {
		return nil, err
	}
	defer done()
	spec := req.ExportRequest
	spec.Query.State = snapshot(ctx, spec.Query.State)
	spec.Query.SQL = spec.Query.State.ExpandAsText(spec.Query.SQL)
	spec.Query.Database = spec.Query.State.ExpandAsText(spec.Query.Database)
	result, err := e.Export(op, spec)
	if err != nil {
		return nil, err
	}
	if req.Destination != "" {
		if err = os.WriteFile(req.Destination, result.Content, 0600); err != nil {
			return result, err
		}
	}
	capture(ctx, req.Capture, req.Database, result)
	return result, nil
}
func (s *service) compare(ctx *endly.Context, req *CompareRequest) (*ExpectResponse, error) {
	e, op, done, err := engine(ctx, req.Database, Assets{})
	if err != nil {
		return nil, err
	}
	defer done()
	spec := req.CompareRequest
	spec.Expected.State = snapshot(ctx, spec.Expected.State)
	spec.Actual.State = snapshot(ctx, spec.Actual.State)
	spec.Expected.SQL = spec.Expected.State.ExpandAsText(spec.Expected.SQL)
	spec.Actual.SQL = spec.Actual.State.ExpandAsText(spec.Actual.SQL)
	spec.Expected.Database = spec.Expected.State.ExpandAsText(spec.Expected.Database)
	spec.Actual.Database = spec.Actual.State.ExpandAsText(spec.Actual.Database)
	report, err := e.Compare(op, spec)
	result := &ExpectResponse{Report: report}
	if err == nil {
		capture(ctx, req.Capture, req.Database, report)
	}
	return result, err
}
