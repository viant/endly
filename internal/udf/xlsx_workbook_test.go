package udf

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestXlsxReaderFixedAndBatch(t *testing.T) {
	content := xlsxTestWorkbook(t)
	fixed, err := NewXlsxReader(map[string]interface{}{"sheet": "Sheet1", "range": "A1:A1"})
	require.NoError(t, err)
	fixedResult, err := fixed(content, nil)
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{
		"byteCount": len(content), "sheet": "Sheet1", "range": "A1:A1",
		"rows": [][]interface{}{{"other sheet"}}, "types": [][]string{{"string"}},
	}, fixedResult)

	read, err := NewXlsxReader(map[string]interface{}{"metadata": true, "ranges": []interface{}{
		map[string]interface{}{"id": "heading", "sheet": "Sheet1", "range": "A1"},
		map[string]interface{}{"id": "table", "sheet": "Selected", "range": "autoFilter"},
		map[string]interface{}{"id": "overlap", "sheet": "Selected", "range": "$B$3:$C$4"},
	}})
	require.NoError(t, err)
	result, err := read(content, nil)
	require.NoError(t, err)
	value := result.(map[string]interface{})
	grids := value["grids"].(map[string]interface{})
	require.Len(t, grids, 3)
	legacy, err := readXlsxGrid(content, "Selected")
	require.NoError(t, err)
	delete(legacy, "byteCount")
	require.Equal(t, legacy, grids["table"], "inventory streaming preserves every legacy grid value and type")
	require.Equal(t, [][]interface{}{{"Key", "Value"}, {"Example", float64(4.25)}}, grids["overlap"].(map[string]interface{})["rows"])
	workbook := value["workbook"].(map[string]interface{})
	require.Equal(t, []string{"Sheet1", "Selected"}, workbook["sheetNames"])
	require.Equal(t, 2, workbook["sheetCount"])
	sheets := workbook["sheets"].(map[string]interface{})
	require.Nil(t, sheets["Sheet1"].(map[string]interface{})["autoFilter"])
	require.Equal(t, "$B$3:$F$5", sheets["Selected"].(map[string]interface{})["autoFilter"])
}

func TestXlsxReaderBatchOptionsAndCumulativeLimits(t *testing.T) {
	grid := map[string]interface{}{"id": "example", "sheet": "Sheet1", "range": "A1:A60000"}
	for _, opts := range []map[string]interface{}{
		{"sheet": "Sheet1", "range": "autoFilter", "ranges": []interface{}{grid}},
		{"sheet": "Sheet1", "range": "autoFilter", "metadata": true},
		{"metadata": "true"},
		{"ranges": []interface{}{grid, grid}},
		{"sheet": "Sheet1", "range": "B5:A2"},
	} {
		_, err := NewXlsxReader(opts)
		require.Error(t, err)
	}
	read, err := NewXlsxReader(map[string]interface{}{"ranges": []interface{}{
		grid, map[string]interface{}{"id": "same-range", "sheet": "Sheet1", "range": "A1:A60000"},
	}})
	require.NoError(t, err)
	_, err = read(xlsxTestWorkbook(t), nil)
	require.ErrorContains(t, err, "ranges and inventory exceed")
}

func TestXlsxReaderStreamsStyleOnlyPadding(t *testing.T) {
	content := xlsxTestWorkbook(t)
	content = mutateXlsxTestPart(t, content, "xl/worksheets/sheet1.xml", "</sheetData>",
		"<row r=\"100001\"><c r=\"XFD100001\" s=\"0\"></c></row></sheetData>")
	read, err := NewXlsxReader(map[string]interface{}{"metadata": true, "ranges": []interface{}{
		map[string]interface{}{"id": "heading", "sheet": "Sheet1", "range": "A1:B1"},
	}})
	require.NoError(t, err)
	result, err := read(content, nil)
	require.NoError(t, err)
	workbook := result.(map[string]interface{})["workbook"].(map[string]interface{})
	sheet := workbook["sheets"].(map[string]interface{})["Sheet1"].(map[string]interface{})
	require.Equal(t, 1, sheet["maxRow"])
	require.Equal(t, map[string]interface{}{"A1": "other sheet"}, sheet["cells"])
}

func TestXlsxReaderCountsDuplicateTextInBatchOutput(t *testing.T) {
	book := excelize.NewFile()
	t.Cleanup(func() { require.NoError(t, book.Close()) })
	value := strings.Repeat("x", 32767)
	for j := 1; j <= 257; j++ {
		require.NoError(t, book.SetCellStr("Sheet1", fmt.Sprintf("A%d", j), value))
	}
	content, err := book.WriteToBuffer()
	require.NoError(t, err)
	read, err := NewXlsxReader(map[string]interface{}{"ranges": []interface{}{
		map[string]interface{}{"id": "first", "sheet": "Sheet1", "range": "A1:A257"},
		map[string]interface{}{"id": "duplicate", "sheet": "Sheet1", "range": "A1:A257"},
	}})
	require.NoError(t, err)
	_, err = read(content.Bytes(), nil)
	require.ErrorContains(t, err, "decoded XLSX cell text exceeds")
}

func xlsxMetadataTestWorkbook(t testing.TB, sheetCount, chartCount int) []byte {
	t.Helper()
	f := excelize.NewFile()
	t.Cleanup(func() { require.NoError(t, f.Close()) })
	require.NoError(t, f.SetSheetName("Sheet1", "Data's Source"))
	for j := 1; j < sheetCount; j++ {
		sheet := fmt.Sprintf("Sheet%02d", j)
		_, err := f.NewSheet(sheet)
		require.NoError(t, err)
		require.NoError(t, f.SetSheetRow(sheet, "A1", &[]interface{}{"Key", "Count"}))
		require.NoError(t, f.SetSheetRow(sheet, "A2", &[]interface{}{"A", j}))
	}
	const source = "Data's Source"
	require.NoError(t, f.SetSheetRow(source, "A1", &[]interface{}{"Key", "Count"}))
	require.NoError(t, f.SetSheetRow(source, "A2", &[]interface{}{"A", 3}))
	require.NoError(t, f.SetSheetRow(source, "A3", &[]interface{}{"B", 4}))
	require.NoError(t, f.SetCellFormula(source, "C1", "SUM(B2:B3)"))
	require.NoError(t, f.SetDefinedName(&excelize.DefinedName{Name: "EntireWorkbook", RefersTo: "'Data''s Source'!$A$1"}))
	require.NoError(t, f.SetDefinedName(&excelize.DefinedName{Name: "LocalName", RefersTo: "Sheet01!$A$1", Scope: "Sheet01"}))
	require.NoError(t, f.SetPanes(source, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"}))
	require.NoError(t, f.SetCellHyperLink(source, "A2", "Sheet01!A1", "Location"))
	require.NoError(t, f.SetCellHyperLink(source, "A3", "https://example.invalid/resource", "External"))
	for j := 1; j <= chartCount; j++ {
		sheet := fmt.Sprintf("Sheet%02d", j)
		require.NoError(t, f.AddChart(sheet, "D4", &excelize.Chart{
			Type: excelize.Col,
			Series: []excelize.ChartSeries{{
				Name: "'Data''s Source'!$B$1", Categories: "'Data''s Source'!$A$2:$A$3",
				Values: "'Data''s Source'!$B$2:$B$3",
			}},
			Legend: excelize.ChartLegend{Position: "bottom"},
		}))
	}
	content, err := f.WriteToBuffer()
	require.NoError(t, err)
	return content.Bytes()
}

func TestXlsxReaderInventory(t *testing.T) {
	content := xlsxMetadataTestWorkbook(t, 3, 2)
	content = mutateXlsxTestPart(t, content, "xl/worksheets/sheet1.xml",
		"<f>SUM(B2:B3)</f></c>", "<f>SUM(B2:B3)</f><v>7</v></c>")
	content = mutateXlsxTestPart(t, content, "xl/worksheets/sheet1.xml",
		"<c r=\"B3\"><v>4</v></c>", "<c r=\"B3\" t=\"e\"><v>#N/A</v></c>")
	reader, err := NewXlsxReader(map[string]interface{}{"metadata": true})
	require.NoError(t, err)
	response, err := reader(content, nil)
	require.NoError(t, err)
	workbook := response.(map[string]interface{})["workbook"].(map[string]interface{})
	require.Equal(t, []string{"Data's Source", "Sheet01", "Sheet02"}, workbook["sheetNames"])
	require.Equal(t, map[string]map[string]string{
		"*": {"EntireWorkbook": "'Data''s Source'!$A$1"}, "Sheet01": {"LocalName": "Sheet01!$A$1"},
	}, workbook["definedNamesBySheet"])
	require.Equal(t, map[string]string{"name": "LocalName", "localSheetId": "1"},
		workbook["definedNames"].([]map[string]interface{})[1]["attributes"])
	sheets := workbook["sheets"].(map[string]interface{})
	data := sheets["Data's Source"].(map[string]interface{})
	require.Equal(t, map[string]string{"ySplit": "1", "topLeftCell": "A2", "activePane": "bottomLeft", "state": "frozen"}, data["pane"])
	require.Equal(t, map[string]string{"A2": "Sheet01!A1", "A3": "https://example.invalid/resource"}, data["hyperlinks"])
	require.Equal(t, map[string]interface{}{"C1": map[string]interface{}{"expression": "SUM(B2:B3)", "cachedValue": "7"}}, data["formulas"])
	require.Equal(t, map[string]string{"B3": "#N/A"}, data["errors"])
	require.Nil(t, data["cells"].(map[string]interface{})["C1"])
	require.Nil(t, data["cells"].(map[string]interface{})["B3"])
	require.Equal(t, 2, workbook["chartCount"])
	chartPath := sheets["Sheet01"].(map[string]interface{})["charts"].([]string)[0]
	chart := workbook["charts"].(map[string]interface{})[chartPath].(map[string]interface{})
	require.Equal(t, true, chart["legendPresent"])
	require.ElementsMatch(t, []string{"'Data''s Source'!$B$1", "'Data''s Source'!$A$2:$A$3", "'Data''s Source'!$B$2:$B$3"}, chart["references"])
	require.Equal(t, []map[string]interface{}{{
		"type": "barChart", "direction": "col", "grouping": "clustered", "label": "'Data''s Source'!$B$1",
		"categories": "'Data''s Source'!$A$2:$A$3", "values": "'Data''s Source'!$B$2:$B$3", "signedFill": false,
	}}, chart["series"])
}

func TestXlsxReaderRejectsExternalDrawing(t *testing.T) {
	content := xlsxMetadataTestWorkbook(t, 2, 1)
	content = mutateXlsxTestPart(t, content, "xl/worksheets/_rels/sheet2.xml.rels",
		"Target=\"../drawings/drawing1.xml\"", "Target=\"https://example.invalid/drawing.xml\" TargetMode=\"External\"")
	reader, err := NewXlsxReader(map[string]interface{}{"metadata": true})
	require.NoError(t, err)
	_, err = reader(content, nil)
	require.ErrorContains(t, err, "invalid or ambiguous XLSX drawing relationship")
}

func BenchmarkXlsxReaderBatch(b *testing.B) {
	content := xlsxMetadataTestWorkbook(b, 73, 28)
	ranges := make([]interface{}, 0, 337)
	for j := 0; j < 337; j++ {
		sheet := fmt.Sprintf("Sheet%02d", j%72+1)
		ranges = append(ranges, map[string]interface{}{"id": fmt.Sprint(j), "sheet": sheet, "range": "A1:B2"})
	}
	reader, err := NewXlsxReader(map[string]interface{}{"ranges": ranges, "metadata": true})
	require.NoError(b, err)
	b.Run("decode", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(content)))
		for b.Loop() {
			result, err := reader(content, nil)
			if err != nil {
				b.Fatal(err)
			}
			observed := result.(map[string]interface{})
			if len(observed["grids"].(map[string]interface{})) != 337 || observed["workbook"].(map[string]interface{})["chartCount"] != 28 {
				b.Fatal("incomplete workbook")
			}
		}
	})
	result, err := reader(content, nil)
	require.NoError(b, err)
	b.Run("serialize", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := json.Marshal(result); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Retained synthetic artifacts can be qualified without adding them to this repository.
func TestXlsxReaderExternalSyntheticFixture(t *testing.T) {
	root := os.Getenv("ENDLY_XLSX_SYNTHETIC_FIXTURES")
	if root == "" {
		t.Skip("optional directory of local synthetic XLSX fixtures")
	}
	files, err := filepath.Glob(filepath.Join(root, "*", "*.xlsx"))
	require.NoError(t, err)
	require.NotEmpty(t, files)
	reader, err := NewXlsxReader(map[string]interface{}{"metadata": true})
	require.NoError(t, err)
	for _, file := range files {
		content, err := os.ReadFile(file)
		require.NoError(t, err)
		result, err := reader(content, nil)
		require.NoError(t, err, filepath.Base(filepath.Dir(file)))
		workbook := result.(map[string]interface{})["workbook"].(map[string]interface{})
		t.Logf("%s %d bytes %d sheets %d charts", filepath.Base(filepath.Dir(file)), len(content), workbook["sheetCount"], workbook["chartCount"])
	}
}
