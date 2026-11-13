package udf

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestXlsxReaderSelectedMetadataText(t *testing.T) {
	content := xlsxTestWorkbook(t)
	content = mutateXlsxTestPart(t, content, "xl/worksheets/sheet1.xml", "</sheetData>",
		`<row r="2"><c r="A2" t="inlineStr"><is><t>雪 café</t></is></c><c r="C2"><v>2.5</v></c><c r="D2" t="b"><v>1</v></c><c r="E2" t="str"><v></v></c></row>`+
			`<row r="3"><c r="A3" t="str"><f>CONCAT(&quot;cached&quot;)</f><v>cached</v></c><c r="B3" t="e"><v>#N/A</v></c><c r="C3"><f>1+1</f><v>2</v></c><c r="D3"><f>1+1</f></c></row>`+
			`<row r="100001"><c r="XFD100001" s="0"></c></row></sheetData>`)
	read := func(options map[string]interface{}) map[string]interface{} {
		reader, err := NewXlsxReader(options)
		require.NoError(t, err)
		response, err := reader(content, nil)
		require.NoError(t, err)
		return response.(map[string]interface{})
	}
	baseline := read(map[string]interface{}{"metadata": true})
	selected := read(map[string]interface{}{"metadata": true, "metadataTextSheets": []interface{}{"Sheet1"}})
	sheets := selected["workbook"].(map[string]interface{})["sheets"].(map[string]interface{})
	sheet := sheets["Sheet1"].(map[string]interface{})
	require.Equal(t, "other sheet 雪 café 2.5 true ", sheet["text"])
	require.NotContains(t, sheets["Selected"].(map[string]interface{}), "text")
	require.Equal(t, 3, sheet["maxRow"], "style-only padding does not become observed data")
	require.Equal(t, 3, sheet["formulaCount"])
	require.Equal(t, "cached", sheet["formulas"].(map[string]interface{})["A3"].(map[string]interface{})["cachedValue"])
	delete(sheet, "text")
	require.Equal(t, baseline, selected, "default metadata stays unchanged")
}

func TestXlsxReaderMetadataTextOptions(t *testing.T) {
	for _, options := range []map[string]interface{}{
		{"metadataTextSheets": []interface{}{"Sheet1"}},
		{"metadata": false, "metadataTextSheets": []interface{}{"Sheet1"}},
		{"metadata": true, "metadataTextSheets": "Sheet1"},
		{"metadata": true, "metadataTextSheets": []interface{}{}},
		{"metadata": true, "metadataTextSheets": []interface{}{nil}},
		{"metadata": true, "metadataTextSheets": []interface{}{" "}},
		{"metadata": true, "metadataTextSheets": []interface{}{"Sheet1", "sheet1"}},
		{"metadata": true, "metadataTextSheets": []interface{}{"K", "k"}},
	} {
		_, err := NewXlsxReader(options)
		require.Error(t, err, "%v", options)
	}
	for _, name := range []string{"Missing", "sheet1"} {
		reader, err := NewXlsxReader(map[string]interface{}{"metadata": true, "metadataTextSheets": []interface{}{name}})
		require.NoError(t, err)
		_, err = reader(xlsxTestWorkbook(t), nil)
		require.ErrorContains(t, err, "missing XLSX worksheet")
	}
}

func TestXlsxReaderMetadataTextRequiresWorksheet(t *testing.T) {
	book := excelize.NewFile()
	t.Cleanup(func() { require.NoError(t, book.Close()) })
	require.NoError(t, book.SetCellStr("Sheet1", "A1", "Point"))
	require.NoError(t, book.SetCellInt("Sheet1", "B1", 1))
	_, err := book.NewSheet("Empty")
	require.NoError(t, err)
	require.NoError(t, book.AddChartSheet("Standalone", &excelize.Chart{
		Type:   excelize.Col,
		Series: []excelize.ChartSeries{{Name: "Sheet1!$A$1", Categories: "Sheet1!$A$1", Values: "Sheet1!$B$1"}},
	}))
	content, err := book.WriteToBuffer()
	require.NoError(t, err)
	reader, err := NewXlsxReader(map[string]interface{}{"metadata": true, "metadataTextSheets": []interface{}{"Empty"}})
	require.NoError(t, err)
	response, err := reader(content.Bytes(), nil)
	require.NoError(t, err)
	workbook := response.(map[string]interface{})["workbook"].(map[string]interface{})
	require.Equal(t, "", workbook["sheets"].(map[string]interface{})["Empty"].(map[string]interface{})["text"])
	reader, err = NewXlsxReader(map[string]interface{}{"metadata": true, "metadataTextSheets": []interface{}{"Standalone"}})
	require.NoError(t, err)
	_, err = reader(content.Bytes(), nil)
	require.ErrorContains(t, err, "chart sheet and has no cell grid")
}

func TestXlsxReaderMetadataTextCountsDuplicateOutput(t *testing.T) {
	book := excelize.NewFile()
	t.Cleanup(func() { require.NoError(t, book.Close()) })
	for row := 1; row <= 257; row++ {
		require.NoError(t, book.SetCellStr("Sheet1", fmt.Sprintf("A%d", row), strings.Repeat("x", 32767)))
	}
	content, err := book.WriteToBuffer()
	require.NoError(t, err)
	reader, err := NewXlsxReader(map[string]interface{}{"metadata": true})
	require.NoError(t, err)
	_, err = reader(content.Bytes(), nil)
	require.NoError(t, err, "ordinary metadata fits the existing text budget")
	reader, err = NewXlsxReader(map[string]interface{}{"metadata": true, "metadataTextSheets": []interface{}{"Sheet1"}})
	require.NoError(t, err)
	_, err = reader(content.Bytes(), nil)
	require.ErrorContains(t, err, `worksheet "Sheet1" metadata text`)
	require.ErrorContains(t, err, "decoded XLSX cell text exceeds")
}

func TestXlsxMetadataTextChargesUnicodeAndSeparators(t *testing.T) {
	values := map[xlsxPosition]xlsxObservedCell{
		{col: 2, row: 1}: {value: "雪", kind: "string"},
		{col: 1, row: 2}: {value: "é", kind: "string"},
	}
	budget := &xlsxBudget{textBytes: xlsxTextLimit - len("雪 é")}
	text, err := xlsxSheetText(values, budget)
	require.NoError(t, err)
	require.Equal(t, "雪 é", text)
	require.Equal(t, xlsxTextLimit, budget.textBytes)
	_, err = xlsxSheetText(values, &xlsxBudget{textBytes: xlsxTextLimit - len("雪 é") + 1})
	require.ErrorContains(t, err, "decoded XLSX cell text exceeds")
}

func BenchmarkXlsxReaderMetadataText(b *testing.B) {
	content := xlsxMetadataTestWorkbook(b, 20, 3)
	for _, selected := range []bool{false, true} {
		name := "default"
		options := map[string]interface{}{"metadata": true}
		if selected {
			name = "selected"
			options["metadataTextSheets"] = []interface{}{"Sheet01", "Sheet02"}
		}
		reader, err := NewXlsxReader(options)
		require.NoError(b, err)
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				result, err := reader(content, nil)
				require.NoError(b, err)
				_, err = json.Marshal(result)
				require.NoError(b, err)
			}
		})
	}
}
