package dbunit

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
	core "github.com/viant/dbunit"
	"github.com/viant/dbunit/fixture"
	"github.com/viant/endly"
	"github.com/viant/endly/model/msg"
	"github.com/viant/mcp-protocol/schema"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
)

//go:embed testdata
var embeddedFixtures embed.FS

func testContext(t *testing.T) (*endly.Context, endly.Service) {
	t.Helper()
	manager := endly.New()
	ctx := manager.NewContext(toolbox.NewContext())
	t.Cleanup(ctx.Close)
	service, err := manager.Service(ServiceID)
	require.NoError(t, err)
	return ctx, service
}
func action(t *testing.T, s endly.Service, ctx *endly.Context, request any) any {
	t.Helper()
	response := s.Run(ctx, request)
	require.NoError(t, response.Err)
	return response.Response
}
func registerDB(t *testing.T, s endly.Service, ctx *endly.Context, name string) {
	t.Helper()
	action(t, s, ctx, &RegisterRequest{Database: name, Driver: "sqlite", DSN: filepath.Join(t.TempDir(), name+".db")})
}
func rowsSource(rows ...map[string]any) fixture.Source { return fixture.Maps(rows) }

func TestSQLiteActionChainSchemaRoundTrip(t *testing.T) {
	ctx, s := testContext(t)
	registerDB(t, s, ctx, "app")
	require.NoError(t, BindFixtures(ctx, "orders", fixture.Embed(&embeddedFixtures, "testdata")))
	ctx.State()["person"] = "alice"
	action(t, s, ctx, &ScriptRequest{Assets: Assets{Fixtures: "orders"}, SchemaApplyRequest: core.SchemaApplyRequest{Database: "app", Sources: []fixture.Source{fixture.File("schema.sql")}}})
	prepared := action(t, s, ctx, &PrepareRequest{Database: "app", Assets: Assets{Fixtures: "orders"}, PrepareRequest: core.PrepareRequest{Datasets: []core.Dataset{{Table: core.Table("users"), Source: fixture.File("users.json"), Operation: core.Replace}}}}).(*core.PrepareResult)
	require.EqualValues(t, 1, prepared.Rows)
	queried := action(t, s, ctx, &QueryRequest{Assets: Assets{Capture: "selected"}, QueryRequest: core.QueryRequest{Database: "app", SQL: "SELECT id,name,active FROM users"}}).(*core.QueryResult)
	require.Equal(t, "alice", queried.Rows[0]["name"])
	state := ctx.State()
	_, found := state.GetValue("selected")
	require.True(t, found)
	expected := core.Expectation{Table: core.Table("users"), Expected: rowsSource(map[string]any{"id": 1, "name": "alice", "active": 0}), Scope: core.Full}
	report := action(t, s, ctx, &ExpectRequest{Database: "app", ExpectRequest: core.ExpectRequest{Expectations: []core.Expectation{expected}}}).(*ExpectResponse)
	require.Zero(t, report.Failed)
	require.Positive(t, report.Passed)
	artifact := action(t, s, ctx, &DumpRequest{SchemaExtractRequest: core.SchemaExtractRequest{Database: "app", Tables: []string{"users"}}, Destination: filepath.Join(t.TempDir(), "schema.sql")}).(*core.SchemaArtifact)
	require.True(t, artifact.Complete)
	registerDB(t, s, ctx, "copy")
	action(t, s, ctx, &ScriptRequest{SchemaApplyRequest: core.SchemaApplyRequest{Database: "copy", Artifact: artifact}})
	action(t, s, ctx, &PrepareRequest{Database: "copy", PrepareRequest: core.PrepareRequest{Datasets: []core.Dataset{{Table: core.Table("users"), Source: expected.Expected}}}})
	action(t, s, ctx, &ExpectRequest{Database: "copy", ExpectRequest: core.ExpectRequest{Expectations: []core.Expectation{expected}}})
	action(t, s, ctx, &SQLRequest{ExecRequest: core.ExecRequest{Database: "copy", Statements: []core.Statement{{SQL: "INSERT INTO users(id) VALUES (?)", Args: []any{2}}}}})
	defaults := action(t, s, ctx, &QueryRequest{QueryRequest: core.QueryRequest{Database: "copy", SQL: "SELECT name,active FROM users WHERE id=2"}}).(*core.QueryResult)
	require.Equal(t, "default", defaults.Rows[0]["name"])
}

func TestFailedAssertionPublishesCompletedCountsOnce(t *testing.T) {
	ctx, s := testContext(t)
	registerDB(t, s, ctx, "app")
	action(t, s, ctx, &ScriptRequest{SchemaApplyRequest: core.SchemaApplyRequest{Database: "app", Statements: []string{"CREATE TABLE users(id INTEGER)", "INSERT INTO users VALUES(1)"}}})
	var reports []*ExpectResponse
	ctx.SetListener(func(event msg.Event) {
		if report, ok := event.Value().(*ExpectResponse); ok {
			reports = append(reports, report)
		}
	})
	response := s.Run(ctx, &ExpectRequest{Database: "app", ExpectRequest: core.ExpectRequest{Expectations: []core.Expectation{{Table: core.Table("users"), Expected: rowsSource(map[string]any{"id": 2})}}}})
	require.Error(t, response.Err)
	var assertion *core.AssertionError
	require.True(t, errors.As(response.Err, &assertion))
	require.Len(t, reports, 1)
	validations := reports[0].Assertion()
	require.Len(t, validations, 1)
	require.Equal(t, reports[0].Failed, validations[0].FailedCount)
	require.Equal(t, reports[0].Passed, validations[0].PassedCount)
	require.Positive(t, validations[0].FailedCount)
	require.Equal(t, validations, reports[0].Assertion())
}

func TestSessionsOwnershipAndCancellation(t *testing.T) {
	first, s := testContext(t)
	second, _ := testContext(t)
	borrowed, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer borrowed.Close()
	require.NoError(t, BindDatabase(first, "shared", borrowed))
	registerDB(t, s, second, "shared")
	holder, err := sessionFor(second)
	require.NoError(t, err)
	owned := holder.databases["shared"].db
	action(t, s, first, &ScriptRequest{SchemaApplyRequest: core.SchemaApplyRequest{Database: "shared", Statements: []string{"CREATE TABLE only_first(id INTEGER)"}}})
	missing := s.Run(second, &QueryRequest{QueryRequest: core.QueryRequest{Database: "shared", SQL: "SELECT * FROM only_first"}})
	require.Error(t, missing.Err)
	require.NoError(t, BindFixtures(first, "assets", fixture.FS(fstest.MapFS{}, ".")))
	response := s.Run(second, &ScriptRequest{Assets: Assets{Fixtures: "assets"}, SchemaApplyRequest: core.SchemaApplyRequest{Database: "shared", Statements: []string{"SELECT 1"}}})
	require.ErrorContains(t, response.Err, "unregistered fixtures")
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	first.SetBackground(canceled)
	require.Error(t, s.Run(first, &QueryRequest{QueryRequest: core.QueryRequest{Database: "shared", SQL: "SELECT 1"}}).Err)
	first.Close()
	first.Close()
	second.Clone().Close()
	second.Close()
	require.NoError(t, borrowed.Ping())
	require.Error(t, owned.Ping())
	require.Error(t, BindDatabase(first, "late", borrowed))
}

func TestRawInlineFixturesExpandExactlyOnce(t *testing.T) {
	ctx, s := testContext(t)
	registerDB(t, s, ctx, "app")
	action(t, s, ctx, &ScriptRequest{SchemaApplyRequest: core.SchemaApplyRequest{Database: "app", Statements: []string{"CREATE TABLE users(id INTEGER, name TEXT)"}}})
	count := 0
	ctx.State()[data.UDFKey].(data.Map)["NextFixture"] = func(_ any, _ data.Map) (any, error) { count++; return "$literal", nil }
	ctx.State()["literal"] = "should not expand"
	raw := map[string]any{"database": "app", "datasets": []any{map[string]any{"table": []any{"users"}, "source": map[string]any{"kind": "maps", "data": []any{map[string]any{"id": 1, "name": "$NextFixture(x)"}}}}}}
	request, err := ctx.AsRequest(ServiceID, "prepare", raw)
	require.NoError(t, err)
	require.Zero(t, count)
	action(t, s, ctx, request)
	require.Equal(t, 1, count)
	result := action(t, s, ctx, &QueryRequest{QueryRequest: core.QueryRequest{Database: "app", SQL: "SELECT name FROM users"}}).(*core.QueryResult)
	require.Equal(t, "$literal", result.Rows[0]["name"])
}

func TestSchemasExposeCoreContract(t *testing.T) {
	_, s := testContext(t)
	for _, name := range s.Actions() {
		route, err := s.Route(name)
		require.NoError(t, err)
		var contract schema.ToolInputSchema
		require.NoError(t, contract.Load(route.RequestProvider()))
		encoded, err := json.Marshal(contract)
		require.NoError(t, err)
		require.Contains(t, string(encoded), "database")
		if name == "prepare" {
			require.Contains(t, string(encoded), "datasets")
			require.Contains(t, string(encoded), "source")
			require.Contains(t, string(encoded), "table")
		}
	}
}

func TestShutdownCancelsBeforeWaitingForActiveOperations(t *testing.T) {
	ctx, _ := testContext(t)
	holder, err := sessionFor(ctx)
	require.NoError(t, err)
	op, done, err := holder.operation(ctx)
	require.NoError(t, err)
	released := make(chan struct{})
	go func() { <-op.Done(); done(); close(released) }()
	ctx.Close()
	<-released
	require.Error(t, op.Err())
}

func TestQueryArgumentsExpandOnceAndSnapshotState(t *testing.T) {
	ctx, s := testContext(t)
	registerDB(t, s, ctx, "app")
	calls := 0
	state := ctx.State()
	state["QueryValue"] = func(any, data.Map) (any, error) { calls++; return "$literal", nil }
	state["literal"] = "must remain literal"
	raw := map[string]any{"database": "app", "sql": "SELECT ? AS value", "args": []any{"$QueryValue()"}}
	request, err := ctx.AsRequest(ServiceID, "query", raw)
	require.NoError(t, err)
	require.Zero(t, calls)
	response := action(t, s, ctx, request).(*core.QueryResult)
	require.Equal(t, 1, calls)
	require.Equal(t, "$literal", response.Rows[0]["value"])
	request = &QueryRequest{QueryRequest: core.QueryRequest{Database: "app", SQL: "SELECT ? AS value", Args: []any{"$QueryValue()"}}}
	response = action(t, s, ctx, request).(*core.QueryResult)
	require.Equal(t, 2, calls)
	require.Equal(t, "$literal", response.Rows[0]["value"])
	action(t, s, ctx, &ScriptRequest{SchemaApplyRequest: core.SchemaApplyRequest{Database: "app", Statements: []string{"CREATE TABLE args(value TEXT)"}}})
	sqlRequest, err := ctx.AsRequest(ServiceID, "sql", map[string]any{"database": "app", "statements": []any{map[string]any{"sql": "INSERT INTO args VALUES (?)", "args": []any{"$QueryValue()"}}, map[string]any{"sql": "INSERT INTO args VALUES ('$literal')"}}})
	require.NoError(t, err)
	require.Equal(t, 2, calls)
	action(t, s, ctx, sqlRequest)
	require.Equal(t, 3, calls)
	inserted := action(t, s, ctx, &QueryRequest{QueryRequest: core.QueryRequest{Database: "app", SQL: "SELECT value FROM args"}}).(*core.QueryResult)
	require.Len(t, inserted.Rows, 2)
	require.Equal(t, "$literal", inserted.Rows[0]["value"])
	require.Equal(t, "$literal", inserted.Rows[1]["value"])
	nested := map[string]any{"list": []any{map[string]any{"value": "original"}}}
	state["nested"] = nested
	copied := snapshot(ctx, nil)
	copied["nested"].(map[string]any)["list"].([]any)[0].(map[string]any)["value"] = "changed"
	require.Equal(t, "original", nested["list"].([]any)[0].(map[string]any)["value"])
}

func TestNestedPresenceExportAndCompare(t *testing.T) {
	ctx, s := testContext(t)
	registerDB(t, s, ctx, "app")
	action(t, s, ctx, &ScriptRequest{SchemaApplyRequest: core.SchemaApplyRequest{Database: "app", Statements: []string{"CREATE TABLE records(id INTEGER PRIMARY KEY, label TEXT DEFAULT 'default', zero INTEGER DEFAULT 9, content TEXT)"}}})
	records := rowsSource(map[string]any{"id": 1, "label": nil, "zero": 0, "content": map[string]any{"empty": []any{}, "null": nil, "nested": map[string]any{"value": 0}}}, map[string]any{"id": 2})
	action(t, s, ctx, &PrepareRequest{Database: "app", PrepareRequest: core.PrepareRequest{Datasets: []core.Dataset{{Table: core.Table("records"), Source: records, Columns: []core.ColumnSpec{{Name: "content", JSON: true}}}}}})
	actual := action(t, s, ctx, &QueryRequest{QueryRequest: core.QueryRequest{Database: "app", SQL: "SELECT * FROM records ORDER BY id"}}).(*core.QueryResult)
	require.Nil(t, actual.Rows[0]["label"])
	require.EqualValues(t, 0, actual.Rows[0]["zero"])
	require.Equal(t, "default", actual.Rows[1]["label"])
	require.EqualValues(t, 9, actual.Rows[1]["zero"])
	var content map[string]any
	require.NoError(t, json.Unmarshal([]byte(actual.Rows[0]["content"].(string)), &content))
	require.Contains(t, content, "null")
	require.Nil(t, content["null"])
	require.Equal(t, []any{}, content["empty"])
	destination := filepath.Join(t.TempDir(), "records.json")
	exported := action(t, s, ctx, &ExportRequest{Database: "app", ExportRequest: core.ExportRequest{Query: core.QueryRequest{SQL: "SELECT * FROM records ORDER BY id"}}, Destination: destination}).(*core.ExportResult)
	require.Equal(t, 2, exported.Rows)
	require.JSONEq(t, string(exported.Content), string(mustRead(t, destination)))
	report := action(t, s, ctx, &CompareRequest{Database: "app", CompareRequest: core.CompareRequest{Expected: core.QueryRequest{SQL: "SELECT * FROM records"}, Actual: core.QueryRequest{SQL: "SELECT * FROM records"}, Match: core.MatchOptions{Keys: []string{"id"}}}}).(*ExpectResponse)
	require.Zero(t, report.Failed)
	require.Positive(t, report.Passed)
}
func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	result, err := os.ReadFile(path)
	require.NoError(t, err)
	return result
}

func TestRegistrationConflictFailureAndReplacement(t *testing.T) {
	ctx, s := testContext(t)
	registerDB(t, s, ctx, "app")
	holder, err := sessionFor(ctx)
	require.NoError(t, err)
	first := holder.databases["app"].db
	conflict := s.Run(ctx, &RegisterRequest{Database: "app", Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "conflict.db")})
	require.ErrorContains(t, conflict.Err, "already registered")
	require.NoError(t, first.Ping())
	replacement := &RegisterRequest{Database: "app", Driver: "sqlite3", DSN: filepath.Join(t.TempDir(), "replacement.db"), Replace: true}
	action(t, s, ctx, replacement)
	require.Error(t, first.Ping())
	require.NoError(t, holder.databases["app"].db.Ping())
	failed := s.Run(ctx, &RegisterRequest{Database: "failed", Driver: "missing-driver", DSN: "unused"})
	require.Error(t, failed.Err)
	require.NotContains(t, holder.databases, "failed")
}

func TestFileAndEmbeddedFixturesHaveEquivalentStateExpansion(t *testing.T) {
	ctx, s := testContext(t)
	registerDB(t, s, ctx, "embedded")
	registerDB(t, s, ctx, "file")
	require.NoError(t, BindFixtures(ctx, "assets", fixture.Embed(&embeddedFixtures, "testdata")))
	root := t.TempDir()
	for _, name := range []string{"schema.sql", "users.json"} {
		content, err := embeddedFixtures.ReadFile("testdata/" + name)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(root, name), content, 0600))
	}
	ctx.State()["person"] = "same"
	for name, assets := range map[string]Assets{"embedded": {Fixtures: "assets"}, "file": {Root: root}} {
		action(t, s, ctx, &ScriptRequest{Assets: assets, SchemaApplyRequest: core.SchemaApplyRequest{Database: name, Sources: []fixture.Source{fixture.File("schema.sql")}}})
		action(t, s, ctx, &PrepareRequest{Database: name, Assets: assets, PrepareRequest: core.PrepareRequest{Datasets: []core.Dataset{{Table: core.Table("users"), Source: fixture.File("users.json")}}}})
	}
	result := action(t, s, ctx, &CompareRequest{Database: "embedded", CompareRequest: core.CompareRequest{Expected: core.QueryRequest{Database: "embedded", SQL: "SELECT * FROM users"}, Actual: core.QueryRequest{Database: "file", SQL: "SELECT * FROM users"}, Match: core.MatchOptions{Keys: []string{"id"}}}}).(*ExpectResponse)
	require.Zero(t, result.Failed)
}

func TestSQLWorkflowTransactionModes(t *testing.T) {
	ctx, s := testContext(t)
	registerDB(t, s, ctx, "app")
	action(t, s, ctx, &ScriptRequest{SchemaApplyRequest: core.SchemaApplyRequest{Database: "app", Statements: []string{"CREATE TABLE modes(id INTEGER PRIMARY KEY)"}}})
	ctx.State()["captureOnSuccess"] = "original"
	for _, test := range []struct {
		mode        core.TransactionMode
		id          int
		committed   int
		transaction string
	}{{core.ExplicitNonAtomic, 1, 1, "nonAtomic"}, {core.PrepareAtomic, 2, 0, "rolledBack"}} {
		request := &SQLRequest{ExecRequest: core.ExecRequest{Database: "app", Statements: []core.Statement{{SQL: "INSERT INTO modes VALUES (?)", Args: []any{test.id}}, {SQL: "INSERT INTO modes VALUES (?)", Args: []any{test.id}}}}, TransactionMode: test.mode, Capture: "captureOnSuccess"}
		result := s.Run(ctx, request)
		require.Error(t, result.Err)
		require.Equal(t, test.transaction, result.Response.(*SQLResponse).Transaction)
		require.Equal(t, "original", ctx.State()["captureOnSuccess"])
		query := action(t, s, ctx, &QueryRequest{QueryRequest: core.QueryRequest{Database: "app", SQL: "SELECT count(*) AS count FROM modes WHERE id=?", Args: []any{test.id}}}).(*core.QueryResult)
		require.EqualValues(t, test.committed, query.Rows[0]["count"])
	}
	for _, mode := range []core.TransactionMode{core.CallerTransaction, core.CaseRollback, "unknown"} {
		result := s.Run(ctx, &SQLRequest{ExecRequest: core.ExecRequest{Database: "app"}, TransactionMode: mode})
		require.ErrorContains(t, result.Err, "requires the Go transaction APIs")
	}
}

func TestExplicitNativeLoadBindingAndSchema(t *testing.T) {
	ctx, s := testContext(t)
	capability := &core.NativeLoadOptions{Transactional: true, Format: "csv", MinRows: 25}
	req := &InitRequest{RegisterRequest: RegisterRequest{Database: "native", Driver: "sqlite", DSN: filepath.Join(t.TempDir(), "native.db"), NativeLoad: capability}, Statements: []string{"CREATE TABLE native_rows(id INTEGER)"}}
	action(t, s, ctx, req)
	holder, err := sessionFor(ctx)
	require.NoError(t, err)
	capability.Format = "changed"
	require.Equal(t, "csv", holder.databases["native"].nativeLoad.Format)
	require.Equal(t, 25, holder.databases["native"].nativeLoad.MinRows)
	failure := s.Run(ctx, &PrepareRequest{Database: "native", TransactionMode: core.ExplicitNonAtomic, PrepareRequest: core.PrepareRequest{Datasets: []core.Dataset{{Table: core.Table("native_rows"), Source: rowsSource(map[string]any{"id": 1}), Load: core.LoadOptions{Strategy: core.BulkRequired}}}}})
	require.ErrorContains(t, failure.Err, "native SQLX loader is not registered")
	borrowed, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer borrowed.Close()
	borrowedCapability := core.NativeLoadOptions{Transactional: false, Format: "json", MinRows: 10}
	require.NoError(t, BindDatabase(ctx, "borrowed-native", borrowed, borrowedCapability))
	borrowedCapability.Format = "changed"
	require.Equal(t, "json", holder.databases["borrowed-native"].nativeLoad.Format)
	require.Error(t, BindDatabase(ctx, "invalid", borrowed, borrowedCapability, borrowedCapability))
	route, err := s.Route("register")
	require.NoError(t, err)
	var contract schema.ToolInputSchema
	require.NoError(t, contract.Load(route.RequestProvider()))
	encoded, err := json.Marshal(contract)
	require.NoError(t, err)
	for _, field := range []string{"nativeLoad", "transactional", "format", "minRows"} {
		require.Contains(t, string(encoded), field)
	}
}

func TestPrepareWorkflowTransactionModes(t *testing.T) {
	ctx, s := testContext(t)
	registerDB(t, s, ctx, "app")
	action(t, s, ctx, &ScriptRequest{SchemaApplyRequest: core.SchemaApplyRequest{Database: "app", Statements: []string{"CREATE TABLE prepare_modes(id INTEGER PRIMARY KEY)"}}})
	for _, test := range []struct {
		mode  core.TransactionMode
		id    int
		count int
	}{{core.PrepareAtomic, 1, 0}, {core.ExplicitNonAtomic, 2, 1}} {
		response := s.Run(ctx, &PrepareRequest{Database: "app", TransactionMode: test.mode, PrepareRequest: core.PrepareRequest{Datasets: []core.Dataset{{Table: core.Table("prepare_modes"), Source: rowsSource(map[string]any{"id": test.id})}, {Table: core.Table("prepare_modes"), Source: rowsSource(map[string]any{"id": test.id})}}}})
		require.Error(t, response.Err)
		actual := action(t, s, ctx, &QueryRequest{QueryRequest: core.QueryRequest{Database: "app", SQL: "SELECT count(*) AS count FROM prepare_modes WHERE id=?", Args: []any{test.id}}}).(*core.QueryResult)
		require.EqualValues(t, test.count, actual.Rows[0]["count"])
	}
	require.ErrorContains(t, s.Run(ctx, &PrepareRequest{Database: "app", TransactionMode: core.CallerTransaction}).Err, "requires the Go transaction APIs")
}

func TestGeneratedSQLUsesSelectedDatabaseCapability(t *testing.T) {
	ctx, s := testContext(t)
	registerDB(t, s, ctx, "app")
	registerDB(t, s, ctx, "other")
	prepare := s.Run(ctx, &PrepareRequest{Database: "app", PrepareRequest: core.PrepareRequest{Datasets: []core.Dataset{{Database: "other", Table: core.Table("rows"), Source: rowsSource(map[string]any{"id": 1})}}}})
	require.ErrorContains(t, prepare.Err, "prepare dataset must use selected database")
	expected := s.Run(ctx, &ExpectRequest{Database: "app", ExpectRequest: core.ExpectRequest{Expectations: []core.Expectation{{Database: "other", Table: core.Table("rows"), Expected: rowsSource(map[string]any{"id": 1})}}}})
	require.ErrorContains(t, expected.Err, "table expectation must use selected database")
	action(t, s, ctx, &ScriptRequest{SchemaApplyRequest: core.SchemaApplyRequest{Database: "app", Statements: []string{"CREATE TABLE rows(id INTEGER)"}}})
	action(t, s, ctx, &PrepareRequest{Database: "app", PrepareRequest: core.PrepareRequest{Datasets: []core.Dataset{{Database: "app", Table: core.Table("rows"), Source: rowsSource(map[string]any{"id": 1})}}}})
	action(t, s, ctx, &ExpectRequest{Database: "app", ExpectRequest: core.ExpectRequest{Expectations: []core.Expectation{{Database: "app", Table: core.Table("rows"), Expected: rowsSource(map[string]any{"id": 1})}}}})
	compared := action(t, s, ctx, &CompareRequest{Database: "app", CompareRequest: core.CompareRequest{Expected: core.QueryRequest{Database: "app", SQL: "SELECT 1 AS id"}, Actual: core.QueryRequest{Database: "other", SQL: "SELECT 1 AS id"}, Match: core.MatchOptions{Keys: []string{"id"}}}}).(*ExpectResponse)
	require.Zero(t, compared.Failed)
}

func TestPrepareUsesCapturedResolvedDatasetsWithoutExpansion(t *testing.T) {
	ctx, s := testContext(t)
	registerDB(t, s, ctx, "app")
	action(t, s, ctx, &ScriptRequest{SchemaApplyRequest: core.SchemaApplyRequest{Database: "app", Statements: []string{"CREATE TABLE composed(id INTEGER PRIMARY KEY, value TEXT)"}}})
	calls := 0
	state := ctx.State()
	state["MustNeverRun"] = func(any, data.Map) (any, error) { calls++; return "wrong", nil }
	state["mapped"] = &struct {
		Datasets []core.Dataset `json:"datasets"`
	}{Datasets: []core.Dataset{{Table: core.Table("composed"), Source: fixture.Resolved([]map[string]any{{"id": 1, "value": "$MustNeverRun()"}})}}}
	request, err := ctx.AsRequest(ServiceID, "prepare", map[string]any{"database": "app", "datasetsFrom": "mapped.datasets"})
	require.NoError(t, err)
	action(t, s, ctx, request)
	require.Zero(t, calls)
	actual := action(t, s, ctx, &QueryRequest{QueryRequest: core.QueryRequest{Database: "app", SQL: "SELECT value FROM composed"}}).(*core.QueryResult)
	require.Equal(t, "$MustNeverRun()", actual.Rows[0]["value"])
	conflict := s.Run(ctx, &PrepareRequest{Database: "app", DatasetsFrom: "mapped.datasets", PrepareRequest: core.PrepareRequest{Datasets: []core.Dataset{{Table: core.Table("composed")}}}})
	require.ErrorContains(t, conflict.Err, "mutually exclusive")
	generatorCalls := 0
	state["generator"] = func() any { generatorCalls++; return []core.Dataset{} }
	for _, path := range []string{"generator", "$mapped.datasets", "missing.datasets", "mapped.datasets++"} {
		_, err := datasetsFrom(state, path)
		require.Error(t, err)
	}
	require.Zero(t, generatorCalls)
}
