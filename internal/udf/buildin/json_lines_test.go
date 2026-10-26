package buildin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/toolbox/data"
)

func jsonLinesFile(t *testing.T, body string) string {
	t.Helper()
	file := filepath.Join(t.TempDir(), "capture.jsonl")
	require.NoError(t, os.WriteFile(file, []byte(body), 0600))
	return file
}
func TestLoadJSONLinesAlwaysReturnsACollection(t *testing.T) {
	for _, body := range []string{"", " \n\t ", `{"method":"read","enabled":false,"zero":0,"nullable":null}`, "{}\n{\"method\":\"read\"}\n\n{\"method\":\"write\"}\n"} {
		t.Run(body[:min(len(body), 16)], func(t *testing.T) {
			result, err := LoadJSONLines(jsonLinesFile(t, body), data.NewMap())
			require.NoError(t, err)
			rows, ok := result.([]interface{})
			require.True(t, ok)
			switch {
			case strings.TrimSpace(body) == "":
				require.Empty(t, rows)
				require.NotNil(t, rows)
			case strings.Contains(body, "enabled"):
				require.Len(t, rows, 1)
				row := rows[0].(map[string]interface{})
				require.Equal(t, false, row["enabled"])
				require.Equal(t, float64(0), row["zero"])
				require.Contains(t, row, "nullable")
				require.Nil(t, row["nullable"])
			default:
				require.Len(t, rows, 3)
				require.Empty(t, rows[0])
			}
		})
	}
}
func TestLoadJSONLinesRejectsMalformedOrNonObjectRecords(t *testing.T) {
	for _, body := range []string{`null`, `[]`, `"text"`, `7`, `{"method":"read"}` + "\n" + `{"broken":`} {
		t.Run(body[:min(len(body), 12)], func(t *testing.T) {
			result, err := LoadJSONLines(jsonLinesFile(t, body), nil)
			require.Error(t, err)
			require.Nil(t, result)
		})
	}
	_, err := LoadJSONLines(jsonLinesFile(t, `{}`)+"-missing", data.NewMap())
	require.Error(t, err)
	for _, value := range []interface{}{nil, "", 17} {
		_, err := LoadJSONLines(value, nil)
		require.Error(t, err)
	}
}
func TestLoadJSONLinesHandlesLargeRecordsAndOwnerRelativeFiles(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "workflow"), 0700))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "captures"), 0700))
	body := `{"payload":"` + strings.Repeat("x", 128*1024) + `"}`
	require.NoError(t, os.WriteFile(filepath.Join(root, "captures", "one.jsonl"), []byte(body), 0600))
	state := data.NewMap()
	state.Put(OwnerURL, "file://"+filepath.Join(root, "workflow", "run.yaml"))
	result, err := LoadJSONLines("captures/one.jsonl", state)
	require.NoError(t, err)
	rows := result.([]interface{})
	require.Len(t, rows, 1)
	require.Len(t, rows[0].(map[string]interface{})["payload"], 128*1024)
}
