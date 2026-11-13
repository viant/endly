package udf

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestXlsxReaderRejectsAmbiguousNativeXML(t *testing.T) {
	content := xlsxTestWorkbook(t)
	for _, tc := range []struct {
		name, from, to, diagnostic string
	}{
		{"formula", "<v>4.25</v>", "<q:f xmlns:q=\"urn:foreign\">1+1</q:f><v>4.25</v>", "ambiguous XLSX cell namespace for f"},
		{"value", "<v>4.25</v>", "<q:v xmlns:q=\"urn:foreign\">-1</q:v>", "ambiguous XLSX cell namespace for v"},
		{"cell", "<c r=\"C4\"><v>4.25</v></c>", "<q:c xmlns:q=\"urn:foreign\" r=\"C4\"><v>4.25</v></q:c>", "ambiguous XLSX row structure at c"},
		{"row", "<row r=\"4\">", "<q:row xmlns:q=\"urn:foreign\" r=\"4\"/><row r=\"4\">", "ambiguous XLSX row structure at row"},
		{"nestedRow", "<sheetData>", "<custom><row r=\"1\"/></custom><sheetData>", "ambiguous XLSX row structure at row"},
		{"nestedCell", "<c r=\"C4\">", "<custom><c r=\"C4\"/></custom><c r=\"C4\">", "ambiguous XLSX row structure at c"},
		{"sheetData", "<sheetData>", "<q:sheetData xmlns:q=\"urn:foreign\"/><sheetData>", "ambiguous XLSX worksheet namespace for sheetData"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			altered := mutateXlsxTestPart(t, content, "xl/worksheets/sheet2.xml", tc.from, tc.to)
			for _, options := range []map[string]interface{}{
				{"sheet": "Selected", "range": "autoFilter"},
				{"metadata": true},
			} {
				read, err := NewXlsxReader(options)
				require.NoError(t, err)
				_, err = read(altered, nil)
				if options["sheet"] != nil && strings.HasPrefix(tc.name, "nested") {
					require.NoError(t, err, "the native cell getter ignores unknown subtrees")
				} else {
					require.ErrorContains(t, err, tc.diagnostic)
				}
			}
		})
	}
}

func TestXlsxReaderBatchOnlyDecodesSelectedValues(t *testing.T) {
	content := xlsxTestWorkbook(t)
	content = mutateXlsxTestPart(t, content, "xl/worksheets/sheet2.xml",
		"<c r=\"H8\" t=\"s\">", "<c r=\"H8\" t=\"unrecognized\"><q:f xmlns:q=\"urn:foreign\">1+1</q:f>")
	content = mutateXlsxTestPart(t, content, "xl/_rels/workbook.xml.rels",
		"</Relationships>",
		"<Relationship Id=\"rId1\" Target=\"xl/nonexistent.xml\" Type=\"irrelevant\"/></Relationships>")
	options := map[string]interface{}{"ranges": []interface{}{
		map[string]interface{}{"id": "selected", "sheet": "Selected", "range": "autoFilter"},
	}}
	read, err := NewXlsxReader(options)
	require.NoError(t, err)
	batch, err := read(content, nil)
	require.NoError(t, err)
	legacy, err := readXlsxGrid(content, "Selected")
	require.NoError(t, err)
	selected := batch.(map[string]interface{})["grids"].(map[string]interface{})["selected"].(map[string]interface{})
	delete(legacy, "byteCount")
	require.Equal(t, legacy, selected)
	require.Equal(t, [][]interface{}{
		{"Key", "Value", "Text", "Flag", "Tail"},
		{"Example", 4.25, "4.25", false, nil},
		{"Last", float64(0), nil, "", nil},
	}, selected["rows"])
}

func TestXlsxReaderIgnoresUnselectedTextBudget(t *testing.T) {
	book := excelize.NewFile()
	t.Cleanup(func() { require.NoError(t, book.Close()) })
	require.NoError(t, book.SetCellStr("Sheet1", "A1", "Keep"))
	text := strings.Repeat("x", 32767)
	for row := 2; row <= 520; row++ {
		require.NoError(t, book.SetCellStr("Sheet1", fmt.Sprintf("A%d", row), text))
	}
	content, err := book.WriteToBuffer()
	require.NoError(t, err)
	read, err := NewXlsxReader(map[string]interface{}{"ranges": []interface{}{
		map[string]interface{}{"id": "selected", "sheet": "Sheet1", "range": "A1"},
	}})
	require.NoError(t, err)
	_, err = read(content.Bytes(), nil)
	require.NoError(t, err)
}

func TestXlsxReaderLimitsFarRightPopulatedRows(t *testing.T) {
	content := xlsxTestWorkbook(t)
	var rows strings.Builder
	for row := 2; row <= 125; row++ {
		fmt.Fprintf(&rows, "<row r=\"%d\"><c r=\"XFD%d\"><v>1</v></c></row>", row, row)
	}
	content = mutateXlsxTestPart(t, content, "xl/worksheets/sheet1.xml", "</sheetData>", rows.String()+"</sheetData>")
	read, err := NewXlsxReader(map[string]interface{}{"metadata": true})
	require.NoError(t, err)
	_, err = read(content, nil)
	require.ErrorContains(t, err, "expanded XLSX streaming work exceeds")
}

func TestXlsxReaderBoundsMergeChecksBeforeGridWork(t *testing.T) {
	content := xlsxTestWorkbook(t)
	var merges strings.Builder
	merges.WriteString("<mergeCells count=\"1100\">")
	for row := 2; row <= 1101; row++ {
		fmt.Fprintf(&merges, "<mergeCell ref=\"B%d:C%d\"/>", row, row)
	}
	merges.WriteString("</mergeCells>")
	content = mutateXlsxTestPart(t, content, "xl/worksheets/sheet1.xml", "</sheetData>", "</sheetData>"+merges.String())
	ranges := make([]interface{}, 1000)
	for i := range ranges {
		ranges[i] = map[string]interface{}{"id": fmt.Sprint(i), "sheet": "Sheet1", "range": "A1"}
	}
	for _, tc := range []struct {
		grids []interface{}
		error string
	}{
		{ranges, "XLSX merge intersection work exceeds"},
		{ranges[:100], ""},
		{[]interface{}{map[string]interface{}{"id": "crosses", "sheet": "Sheet1", "range": "B2"}}, "merged XLSX cells intersect"},
	} {
		read, err := NewXlsxReader(map[string]interface{}{"ranges": tc.grids})
		require.NoError(t, err)
		_, err = read(content, nil)
		if tc.error != "" {
			require.ErrorContains(t, err, tc.error)
		} else {
			require.NoError(t, err)
		}
	}
}

func TestXlsxReaderBoundsNativeGetterMergeChecks(t *testing.T) {
	content := xlsxTestWorkbook(t)
	var data strings.Builder
	for row := 2; row <= 101; row++ {
		fmt.Fprintf(&data, "<row r=\"%d\"><c r=\"A%d\"><v>1</v></c></row>", row, row)
	}
	data.WriteString("</sheetData><mergeCells count=\"11000\">")
	for row := 2; row <= 11001; row++ {
		fmt.Fprintf(&data, "<mergeCell ref=\"B%d:C%d\"/>", row, row)
	}
	data.WriteString("</mergeCells>")
	content = mutateXlsxTestPart(t, content, "xl/worksheets/sheet1.xml", "</sheetData>", data.String())
	read, err := NewXlsxReader(map[string]interface{}{"ranges": []interface{}{
		map[string]interface{}{"id": "column", "sheet": "Sheet1", "range": "A1:A101"},
	}})
	require.NoError(t, err)
	_, err = read(content, nil)
	require.ErrorContains(t, err, "XLSX merge intersection work exceeds")
}

func TestXlsxReaderBoundsRepeatedHyperlinkTargets(t *testing.T) {
	content := xlsxMetadataTestWorkbook(t, 2, 0)
	const sheet = "xl/worksheets/sheet1.xml"
	content = mutateXlsxTestPart(t, content, "xl/worksheets/_rels/sheet1.xml.rels",
		"https://example.invalid/resource", "https://example.invalid/"+strings.Repeat("x", 2<<20))
	var links strings.Builder
	for row := 4; row <= 24; row++ {
		fmt.Fprintf(&links, "<hyperlink ref=\"A%d\" r:id=\"rId1\" location=\"section\"/>", row)
	}
	content = mutateXlsxTestPart(t, content, sheet, "</hyperlinks>", links.String()+"</hyperlinks>")
	read, err := NewXlsxReader(map[string]interface{}{"metadata": true})
	require.NoError(t, err)
	_, err = read(content, nil)
	require.ErrorContains(t, err, "decoded XLSX cell text exceeds")
}

func TestXlsxReaderPreservesFormulaCacheTypes(t *testing.T) {
	content := xlsxTestWorkbook(t)
	content = mutateXlsxTestPart(t, content, "xl/worksheets/sheet1.xml", "</sheetData>", `<row r="2">
<c r="A2"><f>1+1</f></c>
<c r="B2"><f>1+1</f><v></v></c>
<c r="C2" t="str"><f>1+1</f><v>literal</v></c>
<c r="D2" t="n"><f>1+1</f><v>2</v></c>
<c r="E2" t="b"><f>1+1</f><v>0</v></c>
<c r="F2" t="e"><f>1+1</f><v>#N/A</v></c>
</row></sheetData>`)
	read, err := NewXlsxReader(map[string]interface{}{"metadata": true})
	require.NoError(t, err)
	value, err := read(content, nil)
	require.NoError(t, err)
	workbook := value.(map[string]interface{})["workbook"].(map[string]interface{})
	sheet := workbook["sheets"].(map[string]interface{})["Sheet1"].(map[string]interface{})
	formulas := sheet["formulas"].(map[string]interface{})
	cells := sheet["cells"].(map[string]interface{})
	for ref, cached := range map[string]interface{}{
		"A2": nil, "B2": "", "C2": "literal", "D2": float64(2), "E2": false, "F2": "#N/A",
	} {
		require.Equal(t, map[string]interface{}{"expression": "1+1", "cachedValue": cached}, formulas[ref])
		require.Nil(t, cells[ref])
	}
	require.Equal(t, map[string]string{"F2": "#N/A"}, sheet["errors"])
}

func TestXlsxReaderInventoriesChartSheet(t *testing.T) {
	book := excelize.NewFile()
	t.Cleanup(func() { require.NoError(t, book.Close()) })
	require.NoError(t, book.SetCellStr("Sheet1", "A1", "Point"))
	require.NoError(t, book.SetCellInt("Sheet1", "B1", 1))
	require.NoError(t, book.AddChartSheet("Standalone", &excelize.Chart{
		Type:   excelize.Col,
		Series: []excelize.ChartSeries{{Name: "Sheet1!$A$1", Categories: "Sheet1!$A$1", Values: "Sheet1!$B$1"}},
	}))
	content, err := book.WriteToBuffer()
	require.NoError(t, err)
	read, err := NewXlsxReader(map[string]interface{}{"metadata": true})
	require.NoError(t, err)
	value, err := read(content.Bytes(), nil)
	require.NoError(t, err)
	workbook := value.(map[string]interface{})["workbook"].(map[string]interface{})
	require.Equal(t, []string{"Sheet1", "Standalone"}, workbook["sheetNames"])
	sheet := workbook["sheets"].(map[string]interface{})["Standalone"].(map[string]interface{})
	require.Equal(t, "chartsheet", sheet["kind"])
	require.Len(t, sheet["charts"], 1)
	require.Equal(t, 1, workbook["chartCount"])
	read, err = NewXlsxReader(map[string]interface{}{"sheet": "Standalone", "range": "A1"})
	require.NoError(t, err)
	_, err = read(content.Bytes(), nil)
	require.ErrorContains(t, err, "has no cell grid")
}

func TestXlsxReaderRejectsRowsInsideMetadataBeforeSheetData(t *testing.T) {
	content := xlsxTestWorkbook(t)
	const rogueRow = "<row r=\"1\"><c r=\"A1\"><v>999</v></c></row>"
	for name, subtree := range map[string]string{
		"view":    "<sheetViews><sheetView><pane>" + rogueRow + "</pane></sheetView></sheetViews>",
		"link":    "<hyperlinks><hyperlink ref=\"A1\" location=\"A1\">" + rogueRow + "</hyperlink></hyperlinks>",
		"drawing": "<drawing xmlns:r=\"" + xlsxRelationshipsNS + "\" r:id=\"rId1\">" + rogueRow + "</drawing>",
	} {
		t.Run(name, func(t *testing.T) {
			altered := mutateXlsxTestPart(t, content, "xl/worksheets/sheet1.xml", "<sheetData>", subtree+"<sheetData>")
			book, err := excelize.OpenReader(bytes.NewReader(altered))
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, book.Close()) })
			rows, err := book.Rows("Sheet1")
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, rows.Close()) })
			require.True(t, rows.Next())
			columns, err := rows.Columns(excelize.Options{RawCellValue: true})
			require.NoError(t, err)
			require.Equal(t, "999", columns[0], "native Rows sees the injected row before sheetData")

			read, err := NewXlsxReader(map[string]interface{}{"metadata": true})
			require.NoError(t, err)
			_, err = read(altered, nil)
			require.ErrorContains(t, err, "ambiguous XLSX row structure at row")
		})
	}
}

func TestXlsxReaderRejectsTypedContentDiscardedByNativeRows(t *testing.T) {
	content := xlsxTestWorkbook(t)
	for _, tc := range []struct {
		name, cell string
	}{
		{"style", `<c r="C4" s="bad"><v>4.25</v></c>`},
		{"cellMetadata", `<c r="C4" cm="-1"><v>4.25</v></c>`},
		{"valueMetadata", `<c r="C4" vm="bad"><v>4.25</v></c>`},
		{"phonetic", `<c r="C4" ph="bad"><v>4.25</v></c>`},
		{"formulaIndex", `<c r="C4"><f si="bad">1+1</f><v>4.25</v></c>`},
		{"formulaFlag", `<c r="C4"><f aca="bad">1+1</f><v>4.25</v></c>`},
		{"inlineRichText", `<c r="C4" t="inlineStr"><is><r><rPr><sz val="bad"/></rPr><t>kept</t></r></is></c>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			altered := mutateXlsxTestPart(t, content, "xl/worksheets/sheet2.xml", `<c r="C4"><v>4.25</v></c>`, tc.cell)
			if tc.name == "style" {
				book, err := excelize.OpenReader(bytes.NewReader(altered))
				require.NoError(t, err)
				t.Cleanup(func() { require.NoError(t, book.Close()) })
				rows, err := book.Rows("Selected")
				require.NoError(t, err)
				t.Cleanup(func() { require.NoError(t, rows.Close()) })
				var columns []string
				for row := 1; row <= 4; row++ {
					require.True(t, rows.Next())
					columns, err = rows.Columns(excelize.Options{RawCellValue: true})
					require.NoError(t, err)
				}
				require.GreaterOrEqual(t, len(columns), 3)
				require.Empty(t, columns[2], "native Rows silently loses the populated numeric cell")
			}
			read, err := NewXlsxReader(map[string]interface{}{"metadata": true})
			require.NoError(t, err)
			_, err = read(altered, nil)
			require.ErrorContains(t, err, "invalid XLSX typed attribute")
			require.ErrorContains(t, err, "C4")
		})
	}
}

func TestXlsxReaderPreservesTypedAttributeParseError(t *testing.T) {
	content := xlsxTestWorkbook(t)
	read, err := NewXlsxReader(map[string]interface{}{"metadata": true})
	require.NoError(t, err)
	for _, tc := range []struct {
		raw   string
		cause error
	}{
		{"untrusted-attribute-value", strconv.ErrSyntax},
		{"9999999999999999999999999999999999999", strconv.ErrRange},
	} {
		altered := mutateXlsxTestPart(t, content, "xl/worksheets/sheet2.xml", `<c r="C4"><v>4.25</v></c>`,
			fmt.Sprintf(`<c r="C4" s="%s"><v>4.25</v></c>`, tc.raw))
		_, err := read(altered, nil)
		require.ErrorIs(t, err, tc.cause)
		var parsed *strconv.NumError
		require.ErrorAs(t, err, &parsed)
		require.Equal(t, "ParseInt", parsed.Func)
		require.Equal(t, tc.raw, parsed.Num)
		require.ErrorContains(t, err, "invalid XLSX typed attribute c@s in C4: "+tc.cause.Error())
		require.NotContains(t, err.Error(), tc.raw)
	}
}

func TestXlsxReaderBoundsUniqueNativeGetterScans(t *testing.T) {
	content := xlsxTestWorkbook(t)
	var rows strings.Builder
	for row := 2; row <= 2500; row++ {
		fmt.Fprintf(&rows, "<row r=\"%d\"><c r=\"A%d\"><v>1</v></c></row>", row, row)
	}
	content = mutateXlsxTestPart(t, content, "xl/worksheets/sheet1.xml", "</sheetData>", rows.String()+"</sheetData>")
	read, err := NewXlsxReader(map[string]interface{}{"ranges": []interface{}{
		map[string]interface{}{"id": "column", "sheet": "Sheet1", "range": "A1:A2500"},
	}})
	require.NoError(t, err)
	_, err = read(content, nil)
	require.ErrorContains(t, err, "XLSX selected cell lookup work exceeds")

	// Repeated grids reuse the same 1,000 values and fit the lookup budget.
	ranges := make([]interface{}, 6)
	for index := range ranges {
		ranges[index] = map[string]interface{}{"id": fmt.Sprint(index), "sheet": "Sheet1", "range": "A1:A1000"}
	}
	read, err = NewXlsxReader(map[string]interface{}{"ranges": ranges})
	require.NoError(t, err)
	_, err = read(content, nil)
	require.NoError(t, err)
}

func BenchmarkXlsxReaderGridOnlyLookups(b *testing.B) {
	book := excelize.NewFile()
	b.Cleanup(func() { require.NoError(b, book.Close()) })
	for row := 1; row <= 1000; row++ {
		require.NoError(b, book.SetCellInt("Sheet1", fmt.Sprintf("A%d", row), row))
	}
	content, err := book.WriteToBuffer()
	require.NoError(b, err)
	for _, tc := range []struct {
		name      string
		gridCount int
	}{{"unique", 1}, {"overlapping", 6}} {
		b.Run(tc.name, func(b *testing.B) {
			ranges := make([]interface{}, tc.gridCount)
			for index := range ranges {
				ranges[index] = map[string]interface{}{"id": fmt.Sprint(index), "sheet": "Sheet1", "range": "A1:A1000"}
			}
			read, err := NewXlsxReader(map[string]interface{}{"ranges": ranges})
			require.NoError(b, err)
			b.ReportAllocs()
			for b.Loop() {
				if _, err := read(content.Bytes(), nil); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestXlsxReaderRejectsAliasedSelectedParts(t *testing.T) {
	content := xlsxTestWorkbook(t)
	var padding strings.Builder
	for row := 2; row <= 1000; row++ {
		fmt.Fprintf(&padding, "<row r=\"%d\"><c r=\"A%d\" s=\"0\"/></row>", row, row)
	}
	content = mutateXlsxTestPart(t, content, "xl/worksheets/sheet1.xml",
		"<c r=\"A1\" t=\"s\"><v>0</v></c></row></sheetData>",
		"<c r=\"A1\" s=\"0\"/></row>"+padding.String()+"</sheetData>")
	aliased := mutateXlsxTestPart(t, content, "xl/_rels/workbook.xml.rels",
		"Target=\"/xl/worksheets/sheet2.xml\"", "Target=\"/xl/worksheets/sheet1.xml\"")
	for _, options := range []map[string]interface{}{
		{"metadata": true},
		{"ranges": []interface{}{
			map[string]interface{}{"id": "original", "sheet": "Sheet1", "range": "A1"},
			map[string]interface{}{"id": "alias", "sheet": "Selected", "range": "A1"},
		}},
	} {
		read, err := NewXlsxReader(options)
		require.NoError(t, err)
		_, err = read(aliased, nil)
		require.ErrorContains(t, err, "resolve to the same XLSX package part")
	}
	read, err := NewXlsxReader(map[string]interface{}{"sheet": "Sheet1", "range": "A1"})
	require.NoError(t, err)
	_, err = read(aliased, nil)
	require.NoError(t, err, "an unselected alias does not affect a single selected grid")

	relativeAlias := mutateXlsxTestPart(t, content, "xl/_rels/workbook.xml.rels",
		"Target=\"/xl/worksheets/sheet2.xml\"", "Target=\"worksheets/../worksheets/sheet1.xml\"")
	relativeAlias = mutateXlsxTestPart(t, relativeAlias, "xl/worksheets/sheet1.xml",
		"<row r=\"2\">", "<row r=\"0\">")
	read, err = NewXlsxReader(map[string]interface{}{"metadata": true})
	require.NoError(t, err)
	_, err = read(relativeAlias, nil)
	require.ErrorContains(t, err, "resolve to the same XLSX package part", "canonical aliases fail before worksheet parsing")
}

func TestXlsxReaderBoundsRepeatedRichTextMaterialization(t *testing.T) {
	book := excelize.NewFile()
	t.Cleanup(func() { require.NoError(t, book.Close()) })
	_, err := book.NewSheet("Second")
	require.NoError(t, err)
	for _, sheet := range []string{"Sheet1", "Second"} {
		for row := 1; row <= 101; row++ {
			require.NoError(t, book.SetCellStr(sheet, fmt.Sprintf("A%d", row), "seed"))
		}
		require.NoError(t, book.AutoFilter(sheet, "A1:A101", nil))
	}
	raw, err := book.WriteToBuffer()
	require.NoError(t, err)
	const sharedStrings = "xl/sharedStrings.xml"
	const seed = "<si><t>seed</t></si>"
	smallOutput := "<si>" + strings.Repeat("<r><t></t></r>", 10000) + "<r><t>x</t></r></si>"
	content := mutateXlsxTestPart(t, raw.Bytes(), sharedStrings, seed, smallOutput)
	for _, options := range []map[string]interface{}{
		{"metadata": true},
		{"ranges": []interface{}{
			map[string]interface{}{"id": "first", "sheet": "Sheet1", "range": "A1:A101"},
			map[string]interface{}{"id": "second", "sheet": "Second", "range": "A1:A101"},
		}},
	} {
		read, err := NewXlsxReader(options)
		require.NoError(t, err)
		_, err = read(content, nil)
		require.ErrorContains(t, err, "XLSX shared string materialization work exceeds")
	}
	read, err := NewXlsxReader(map[string]interface{}{"sheet": "Sheet1", "range": "autoFilter"})
	require.NoError(t, err)
	value, err := read(content, nil)
	require.NoError(t, err)
	require.Equal(t, []interface{}{"x"}, value.(map[string]interface{})["rows"].([][]interface{})[100])
	overlap, err := NewXlsxReader(map[string]interface{}{"ranges": []interface{}{
		map[string]interface{}{"id": "first", "sheet": "Sheet1", "range": "A1:A101"},
		map[string]interface{}{"id": "repeat", "sheet": "Sheet1", "range": "A1:A101"},
	}})
	require.NoError(t, err)
	_, err = overlap(content, nil)
	require.NoError(t, err, "overlapping grids reuse prior native observations")
	tooManyRuns := mutateXlsxTestPart(t, raw.Bytes(), sharedStrings, seed,
		"<si>"+strings.Repeat("<r><t></t></r>", 20000)+"<r><t>x</t></r></si>")
	_, err = read(tooManyRuns, nil)
	require.ErrorContains(t, err, "XLSX shared string materialization work exceeds")
}
