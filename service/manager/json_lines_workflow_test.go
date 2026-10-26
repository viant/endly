package manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	_ "github.com/viant/endly/internal/udf"
)

type jsonLinesInspection struct {
	Rows    interface{}
	Count   int
	Matches int
}

func TestNativeWorkflowJSONLinesZeroOneAndMany(t *testing.T) {
	cases := []struct {
		name, body     string
		count, matches int
	}{{"empty", "", 0, 0}, {"single", `{"method":"read","enabled":false,"zero":0}`, 1, 1}, {"many", "{\"method\":\"read\"}\n{\"method\":\"write\"}\n{\"method\":\"read\"}\n", 3, 2}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			file := filepath.Join(root, "capture.jsonl")
			require.NoError(t, os.WriteFile(file, []byte(tc.body), 0600))
			var captured jsonLinesInspection
			service := New(func() endly.Manager {
				m := endly.New()
				s := endly.NewAbstractService("jsonLinesCapture")
				s.Register(&endly.Route{Action: "inspect", RequestProvider: func() interface{} { return &jsonLinesInspection{} }, ResponseProvider: func() interface{} { return &struct{}{} }, Handler: func(_ *endly.Context, r interface{}) (interface{}, error) {
					captured = *r.(*jsonLinesInspection)
					return &struct{}{}, nil
				}})
				m.Register(s)
				return m
			})
			defer service.Shutdown(context.Background())
			session, err := service.Open(context.Background(), &OpenRequest{})
			require.NoError(t, err)
			yaml := fmt.Sprintf(`pipeline:
  inspect:
    action: jsonLinesCapture:inspect
    init:
      - name: traffic
        value: $LoadJSONLines('%s')
      - name: selected
        value: $Select('traffic[method = read]')
      - name: total
        value: $Len($traffic)
      - name: matched
        value: $Len($selected)
    Rows: $traffic
    Count: $total
    Matches: $matched
`, file)
			_, err = service.LoadWorkflow(context.Background(), &LoadWorkflowRequest{SessionID: session.SessionID, Alias: "root", URL: filepath.Join(root, "run.yaml"), Format: "yaml", Content: yaml})
			require.NoError(t, err)
			op, err := service.StartWorkflow(&RunWorkflowRequest{SessionID: session.SessionID, Workflow: "root"})
			require.NoError(t, err)
			op, err = service.WaitOperation(context.Background(), session.SessionID, op.ID)
			require.NoError(t, err)
			require.Equal(t, OperationSucceeded, op.Status, op.Error)
			rows, ok := captured.Rows.([]interface{})
			require.True(t, ok, "the native request must retain collection type")
			require.Len(t, rows, tc.count)
			require.Equal(t, tc.count, captured.Count)
			require.Equal(t, tc.matches, captured.Matches)
			if tc.name == "single" {
				row := rows[0].(map[string]interface{})
				require.Equal(t, false, row["enabled"])
				require.EqualValues(t, 0, row["zero"])
			}
		})
	}
}
