package udf

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func xlsxTestWorkbook(t *testing.T) []byte {
	t.Helper()
	f := excelize.NewFile()
	t.Cleanup(func() { require.NoError(t, f.Close()) })
	_, err := f.NewSheet("Selected")
	require.NoError(t, err)
	require.NoError(t, f.SetCellStr("Sheet1", "A1", "other sheet"))
	require.NoError(t, f.SetSheetRow("Selected", "B3", &[]interface{}{"Key", "Value", "Text", "Flag", "Tail"}))
	require.NoError(t, f.SetSheetRow("Selected", "B4", &[]interface{}{"Example", 4.25, "4.25", false}))
	require.NoError(t, f.SetCellStr("Selected", "E5", ""))
	require.NoError(t, f.SetCellDefault("Selected", "F5", ""))
	require.NoError(t, f.SetCellInt("Selected", "C5", 0))
	require.NoError(t, f.SetCellStr("Selected", "B5", "Last"))
	require.NoError(t, f.SetCellStr("Selected", "H8", "outside range"))
	require.NoError(t, f.AutoFilter("Selected", "B3:F5", nil))
	b, err := f.WriteToBuffer()
	require.NoError(t, err)
	return b.Bytes()
}

func mutateXlsxTestPart(t *testing.T, content []byte, part, from, to string) []byte {
	t.Helper()
	z, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	require.NoError(t, err)
	buf := &bytes.Buffer{}
	w := zip.NewWriter(buf)
	changed := false
	for _, entry := range z.File {
		if entry.Name != part {
			require.NoError(t, w.Copy(entry))
			continue
		}
		r, err := entry.Open()
		require.NoError(t, err)
		b, err := io.ReadAll(r)
		require.NoError(t, err)
		require.NoError(t, r.Close())
		require.Contains(t, string(b), from)
		target, err := w.Create(entry.Name)
		require.NoError(t, err)
		_, err = io.WriteString(target, strings.Replace(string(b), from, to, 1))
		require.NoError(t, err)
		changed = true
	}
	require.True(t, changed)
	require.NoError(t, w.Close())
	return buf.Bytes()
}

func TestXlsxReaderObservedGrid(t *testing.T) {
	content := xlsxTestWorkbook(t)
	reader, err := NewXlsxReader(map[string]interface{}{"sheet": "Selected", "range": "autoFilter"})
	require.NoError(t, err)
	result, err := reader(content, nil)
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{
		"sheet": "Selected", "range": "$B$3:$F$5", "byteCount": len(content),
		"rows": [][]interface{}{
			{"Key", "Value", "Text", "Flag", "Tail"},
			{"Example", 4.25, "4.25", false, nil},
			{"Last", float64(0), nil, "", nil},
		},
		"types": [][]string{
			{"string", "string", "string", "string", "string"},
			{"string", "number", "string", "boolean", "blank"},
			{"string", "number", "blank", "string", "blank"},
		},
	}, result)
}

func TestXlsxReaderRejectsUnsupportedInput(t *testing.T) {
	content := xlsxTestWorkbook(t)
	const sheet = "xl/worksheets/sheet2.xml"
	reader, err := NewXlsxReader(map[string]interface{}{"sheet": "Selected", "range": "autoFilter"})
	require.NoError(t, err)
	_, err = reader([]byte("not a zip archive"), nil)
	require.ErrorContains(t, err, "open XLSX workbook")
	_, err = readXlsxGrid(content, "selected")
	require.ErrorContains(t, err, "worksheet")
	_, err = readXlsxGrid(content, "Sheet1")
	require.ErrorContains(t, err, "AutoFilter")
	for _, tc := range []struct{ part, from, to, diagnostic string }{
		{sheet, `autoFilter ref="$B$3:$F$5"`, `autoFilter ref="A1:XFD1048576"`, "range exceeds"},
		{sheet, `autoFilter ref="$B$3:$F$5"`, `autoFilter ref="F5:B3"`, "invalid XLSX range"},
		{sheet, `<row r="3">`, `<row r="-1">`, "invalid or out-of-order XLSX row"},
		{sheet, `r="B4"`, `r="XFD4"`, "invalid or out-of-order XLSX worksheet cell"},
		{sheet, `r="C4"`, `r="C5"`, "invalid or out-of-order XLSX worksheet cell"},
		{sheet, `<row r="8"><c r="H8"`, `<row r="100001"><c r="H100001"`, "expanded XLSX worksheet exceeds"},
		{sheet, `<v>4.25</v>`, `<f/><v>4.25</v>`, "formula XLSX cell C4"},
		{sheet, `<v>4.25</v>`, `<v>NaN</v>`, "invalid XLSX cell value in C4"},
		{sheet, `<c r="C4"><v>4.25</v>`, `<c r="C4" t="e"><v>#DIV/0!</v>`, "error XLSX cell C4"},
		{"xl/_rels/workbook.xml.rels", `Target="/xl/worksheets/sheet2.xml"`, `TargetMode="External" Target="/xl/worksheets/sheet2.xml"`, "invalid or ambiguous XLSX worksheet relationship"},
		{"xl/_rels/workbook.xml.rels", `Target="/xl/worksheets/sheet2.xml"`, `Target="/xl/worksheets/sheet%32.xml"`, "invalid XLSX worksheet target"},
	} {
		t.Run(tc.diagnostic, func(t *testing.T) {
			altered := mutateXlsxTestPart(t, content, tc.part, tc.from, tc.to)
			_, err := reader(altered, nil)
			require.ErrorContains(t, err, tc.diagnostic)
		})
	}
}

func TestXlsxReaderRejectsUnsignedZip64Overflow(t *testing.T) {
	buf := &bytes.Buffer{}
	w := zip.NewWriter(buf)
	raw, err := w.CreateRaw(&zip.FileHeader{Name: "large", Method: zip.Store, UncompressedSize64: 1 << 63, CompressedSize64: 1})
	require.NoError(t, err)
	_, err = raw.Write([]byte("x"))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	_, err = readXlsxGrid(buf.Bytes(), "Selected")
	require.ErrorContains(t, err, "unpacked XLSX exceeds")
}

func TestXlsxReaderBoundsRepeatedSharedText(t *testing.T) {
	f := excelize.NewFile()
	t.Cleanup(func() { require.NoError(t, f.Close()) })
	const maxExcelText = 32_767
	repeated := strings.Repeat("x", maxExcelText)
	const lastRow = xlsxTextLimit/maxExcelText + 1
	for row := 1; row <= lastRow; row++ {
		ref, err := excelize.CoordinatesToCellName(1, row)
		require.NoError(t, err)
		require.NoError(t, f.SetCellStr("Sheet1", ref, repeated))
	}
	end, err := excelize.CoordinatesToCellName(1, lastRow)
	require.NoError(t, err)
	require.NoError(t, f.AutoFilter("Sheet1", "A1:"+end, nil))
	content, err := f.WriteToBuffer()
	require.NoError(t, err)
	_, err = readXlsxGrid(content.Bytes(), "Sheet1")
	require.ErrorContains(t, err, "decoded XLSX cell text exceeds")
}
