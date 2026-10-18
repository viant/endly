package table

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestExportSemanticHeadersAndExplicitNames(t *testing.T) {
	e, err := NewExporter(`<table><thead><tr><th>Name</th><th>Count</th></tr></thead><tbody><tr><th scope="row">Aurora</th><td>12</td></tr><tr><td>Cedar</td><td>7</td></tr></tbody></table>`)
	require.NoError(t, err)
	objects, err := e.Export(nil, "objects")
	require.NoError(t, err)
	require.Equal(t, []map[string]interface{}{{"Name": "Aurora", "Count": "12"}, {"Name": "Cedar", "Count": "7"}}, objects)
	objects, err = e.Export([]string{"_", "count"}, "objects")
	require.NoError(t, err)
	require.Equal(t, []map[string]interface{}{{"count": "12"}, {"count": "7"}}, objects)
	csv, err := e.Export(nil, "csv")
	require.NoError(t, err)
	require.Equal(t, "Name,Count\nAurora,12\nCedar,7\n", csv)
}
func TestExportEmptyAndMalformedTables(t *testing.T) {
	for _, html := range []string{`<table></table>`, `<table><thead><tr><th>Name</th></tr></thead></table>`} {
		e, err := NewExporter(html)
		require.NoError(t, err)
		got, err := e.Export(nil, "json")
		require.NoError(t, err)
		require.Equal(t, "[]", got)
	}
	for _, html := range []string{`<table><tr><th>A</th><th>B</th></tr><tr><td>one</td></tr></table>`, `<table><tr><th>A</th><th>A</th></tr><tr><td>one</td><td>two</td></tr></table>`} {
		e, err := NewExporter(html)
		require.NoError(t, err)
		_, err = e.Export(nil, "objects")
		require.Error(t, err)
	}
}
func TestNestedTablesDoNotAddRowsOrCellText(t *testing.T) {
	e, err := NewExporter(`<table><tr><th>Name</th></tr><tr><td>Aurora<table><tr><td>Nested</td></tr></table></td></tr></table>`)
	require.NoError(t, err)
	got, err := e.Export(nil, "objects")
	require.NoError(t, err)
	require.Equal(t, []map[string]interface{}{{"Name": "Aurora"}}, got)
}
