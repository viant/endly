package manager

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	_ "github.com/viant/endly/internal/udf"
	_ "github.com/viant/endly/service/system/storage"
	_ "github.com/viant/endly/service/testing/validator"
	"github.com/xuri/excelize/v2"
)

func TestNativeWorkflowXlsxInventory(t *testing.T) {
	root := t.TempDir()
	filename := filepath.Join(root, "inventory.xlsx")
	book := excelize.NewFile()
	require.NoError(t, book.SetCellStr("Sheet1", "A4", "Inventory"))
	require.NoError(t, book.SetCellFormula("Sheet1", "A5", "1+1"))
	require.NoError(t, book.SaveAs(filename))
	require.NoError(t, book.Close())

	ctx := context.Background()
	service := New(endly.New)
	defer service.Shutdown(ctx)
	session, err := service.Open(ctx, &OpenRequest{})
	require.NoError(t, err)
	for _, tc := range []struct {
		name, ref string
	}{
		{"success", "A4:A4"},
		{"formula", "A5:A5"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			yaml := fmt.Sprintf(`pipeline:
  register:
    action: udf:register
    udfs:
      - id: NativeXlsx_%s
        provider: XlsxReader
        params:
          - metadata: true
            ranges:
              - {id: heading, sheet: Sheet1, range: %s}
  read:
    action: storage:download
    source: {URL: '%s'}
    udf: NativeXlsx_%s
    destKey: nativeXlsx
  verify:
    action: validator:assert
    actual: nativeXlsx.grids.heading
    expect:
      sheet: Sheet1
      range: A4:A4
      rows: [[Inventory]]
      types: [[string]]
  verify-metadata:
    action: validator:assert
    actual: nativeXlsx.workbook
    expect:
      sheetNames: [Sheet1]
      sheetCount: 1
      chartCount: 0
      sheets:
        Sheet1:
          formulaCount: 1
          formulas:
            A5:
              expression: 1+1
`, tc.name, tc.ref, filename, tc.name)
			_, err := service.LoadWorkflow(ctx, &LoadWorkflowRequest{
				SessionID: session.SessionID, Alias: tc.name,
				URL: filepath.Join(root, tc.name+".yaml"), Format: "yaml", Content: yaml,
			})
			require.NoError(t, err)
			op, err := service.StartWorkflow(&RunWorkflowRequest{SessionID: session.SessionID, Workflow: tc.name})
			require.NoError(t, err)
			op, err = service.WaitOperation(ctx, session.SessionID, op.ID)
			require.NoError(t, err)
			if tc.name == "success" {
				require.Equal(t, OperationSucceeded, op.Status, op.Error)
				require.Greater(t, op.PassedAssertions, 0)
				require.Zero(t, op.FailedAssertions)
			} else {
				require.Equal(t, OperationFailed, op.Status)
				require.Contains(t, op.Error, "failed to transform with NativeXlsx_formula udf")
				require.Contains(t, op.Error, "formula XLSX cell A5 is unsupported")
				require.Zero(t, op.PassedAssertions)
			}
		})
	}
}
