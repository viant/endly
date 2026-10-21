package dbunit_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	manager "github.com/viant/endly/service/manager"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
)

type hydrationManager struct {
	endly.Manager
	once, unexpected *atomic.Int64
}

func (m *hydrationManager) NewContext(base toolbox.Context) *endly.Context {
	ctx := m.Manager.NewContext(base)
	state := ctx.State()
	state["Once"] = func(any, data.Map) (any, error) { m.once.Add(1); return "$NotAgain()", nil }
	state["NotAgain"] = func(any, data.Map) (any, error) { m.unexpected.Add(1); return "expanded twice", nil }
	return ctx
}

func TestHydrationMappingWorkflowAcrossDatabasesAndParents(t *testing.T) {
	firstPath := filepath.Join(t.TempDir(), "first.db")
	secondPath := filepath.Join(t.TempDir(), "second.db")
	var once, unexpected atomic.Int64
	factory := func() endly.Manager {
		return &hydrationManager{Manager: endly.New(), once: &once, unexpected: &unexpected}
	}
	runtime := manager.New(factory)
	defer runtime.Shutdown(context.Background())
	session, err := runtime.Open(context.Background(), &manager.OpenRequest{})
	require.NoError(t, err)
	content := fmt.Sprintf(`init:
  created:
    keep: original
pipeline:
  first:
    action: dbunit:init
    database: first
    driver: sqlite
    dsn: %s
    statements:
      - CREATE TABLE parents(id INTEGER PRIMARY KEY, name TEXT)
      - INSERT INTO parents VALUES(10, 'seed-first')
  second:
    action: dbunit:init
    database: second
    driver: sqlite
    dsn: %s
    statements:
      - CREATE TABLE children(id INTEGER PRIMARY KEY, parent_id INTEGER, name TEXT)
      - INSERT INTO children VALUES(1000, 0, 'seed-second')
  generate:
    action: dbunit:hydrate
    database: first
    capture: generated
    sequences:
      - name: parentIDs
        seed:
          database: first
          sql: SELECT MAX(id)+1 FROM parents
      - name: childIDs
        seed:
          database: second
          sql: SELECT MAX(id)+1 FROM children
    tables:
      - name: parents
        table: [parents]
        tagField: ['!tag']
        source:
          kind: maps
          data:
            - {'!tag': p1, name: '$Once()'}
            - {'!tag': p2, name: '$Once()'}
        generate:
          - field: [id]
            sequence: parentIDs
        captures:
          - field: [id]
            path: created.parentIDs
            append: true
      - name: children
        table: [children]
        source:
          kind: maps
          data:
            - {parentTag: p1, name: child-1}
            - {parentTag: p2, name: child-2}
        generate:
          - field: [id]
            sequence: childIDs
        references:
          - field: [parent_id]
            table: parents
            tag: $row.parentTag
            from: [id]
  parentMap:
    action: dbunit:map
    database: first
    sourceFrom: generated.tables[0].source
    capture: mappedParents
    mapping: &parentMapping
      name: parents
      tables:
        - name: parents
          table: [parents]
          keys: [id]
          columns:
            - {name: id, required: true}
            - {name: name}
  childMap:
    action: dbunit:map
    database: second
    sourceFrom: generated.tables[1].source
    capture: mappedChildren
    mapping: &childMapping
      name: children
      tables:
        - name: children
          table: [children]
          keys: [id]
          columns:
            - {name: id, required: true}
            - {name: parent_id, required: true}
            - {name: name}
  parentPrepare:
    action: dbunit:prepare
    database: first
    datasetsFrom: mappedParents.datasets
  childPrepare:
    action: dbunit:prepare
    database: second
    datasetsFrom: mappedChildren.datasets
  continueSequences:
    action: dbunit:sequence
    database: first
    capture: next
    sequences:
      - {name: parentIDs, startFrom: generated.sequences.parentIDs.next}
      - {name: childIDs, startFrom: generated.sequences.childIDs.next}
  generateMore:
    action: dbunit:hydrate
    database: first
    capture: more
    sequences:
      - {name: parentIDs, startFrom: next.sequences.parentIDs.next}
      - {name: childIDs, startFrom: next.sequences.childIDs.next}
    tables:
      - name: parents
        table: [parents]
        tagField: ['!tag']
        source:
          kind: maps
          data:
            - {'!tag': p3, name: '$Once()'}
        generate:
          - {field: [id], sequence: parentIDs}
      - name: children
        table: [children]
        source:
          kind: maps
          data:
            - {parentTag: p3, name: child-3}
        generate:
          - {field: [id], sequence: childIDs}
        references:
          - {field: [parent_id], table: parents, tag: '$row.parentTag', from: [id]}
  moreParents:
    action: dbunit:map
    database: first
    sourceFrom: more.tables[0].source
    capture: mappedMoreParents
    mapping: *parentMapping
  moreChildren:
    action: dbunit:map
    database: second
    sourceFrom: more.tables[1].source
    capture: mappedMoreChildren
    mapping: *childMapping
  prepareMoreParents:
    action: dbunit:prepare
    database: first
    datasetsFrom: mappedMoreParents.datasets
  prepareMoreChildren:
    action: dbunit:prepare
    database: second
    datasetsFrom: mappedMoreChildren.datasets
  verifyParents:
    action: dbunit:expect
    database: first
    expectations:
      - table: [parents]
        scope: keys
        match: {keys: [id]}
        expected:
          kind: maps
          data: [{id: 11}, {id: 12}, {id: 13}]
  verifyChildren:
    action: dbunit:expect
    database: second
    expectations:
      - table: [children]
        scope: keys
        match: {keys: [id]}
        expected:
          kind: maps
          data:
            - {id: 1001, parent_id: 11}
            - {id: 1002, parent_id: 12}
            - {id: 1003, parent_id: 13}
`, firstPath, secondPath)
	_, err = runtime.LoadWorkflow(context.Background(), &manager.LoadWorkflowRequest{SessionID: session.SessionID, URL: filepath.Join(t.TempDir(), "hydration.yaml"), Content: content, Format: "yaml"})
	require.NoError(t, err)
	operation, err := runtime.StartWorkflow(&manager.RunWorkflowRequest{SessionID: session.SessionID, Workflow: "hydration", Tasks: "*"})
	require.NoError(t, err)
	operation, err = runtime.WaitOperation(context.Background(), session.SessionID, operation.ID)
	require.NoError(t, err)
	require.Equal(t, manager.OperationSucceeded, operation.Status, operation.Error)
	require.Zero(t, operation.FailedAssertions)
	require.EqualValues(t, 3, once.Load())
	require.Zero(t, unexpected.Load())
	capture, err := runtime.InspectContext(&manager.InspectContextRequest{SessionID: session.SessionID, Path: "created.keep"})
	require.NoError(t, err)
	require.Equal(t, "original", capture.Value)
	ids, err := runtime.InspectContext(&manager.InspectContextRequest{SessionID: session.SessionID, Path: "created.parentIDs"})
	require.NoError(t, err)
	require.EqualValues(t, []any{int64(11), int64(12)}, ids.Value)
	db, err := sql.Open("sqlite", firstPath)
	require.NoError(t, err)
	defer db.Close()
	rows, err := db.Query("SELECT id,name FROM parents WHERE id>10 ORDER BY id")
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var id int64
		var name string
		require.NoError(t, rows.Scan(&id, &name))
		require.Equal(t, "$NotAgain()", name)
	}
	require.NoError(t, rows.Err())
}
