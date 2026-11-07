package udf

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/viant/assertly"
	"github.com/viant/toolbox"
)

func TestNewCsvReader(t *testing.T) {

	resultMap := map[string]interface{}{
		"name":          "Bob",
		"facoriteColor": "red",
	}

	rBytes, _ := json.Marshal(resultMap)

	fmt.Printf("!%v!", string(rBytes))

	useCases := []struct {
		description string
		header      string
		delimiter   string
		data        string
		expected    string
		hasError    bool
	}{
		{
			description: "simple csv",
			header:      "first_name,last_name,username",
			data: `"Rob","Pike",rob
Ken,Thompson,ken
`,
			expected: `[{"first_name":"Rob"},{"first_name":"Ken"}]`,
		},
		{
			description: "no header",
			header:      "",
			hasError:    true,
		},
		{
			description: "invalid header or delimiter",
			header:      "id;name",
			hasError:    true,
		},
	}
	for _, useCase := range useCases {
		udfFunc, err := NewCsvReader(useCase.header, useCase.delimiter)
		if useCase.hasError {
			assert.NotNil(t, err, useCase.description)
			continue
		}
		if !assert.Nil(t, err) {
			continue
		}
		transformed, err := udfFunc(useCase.data, nil)
		if !assert.Nil(t, err) {
			continue
		}
		actual := toolbox.AsString(transformed)
		assertly.AssertValues(t, useCase.expected, actual, useCase.description)
	}

}

func TestCsvReaderRows(t *testing.T) {
	read, err := NewCsvReader(map[string]interface{}{"output": "rows"})
	require.NoError(t, err)

	rows, err := read([]byte("\ufeff\"Name, display\",Value,Empty\r\n\"Two\r\nlines\",\"a\"\"b\",\r\n\ufeffinside,1\r\n"), nil)
	require.NoError(t, err)
	require.Equal(t, [][]string{
		{"Name, display", "Value", "Empty"},
		{"Two\nlines", `a"b`, ""},
		{"\ufeffinside", "1"},
	}, rows)

	rows, err = read("", nil)
	require.NoError(t, err)
	require.Equal(t, [][]string{}, rows)

	rows, err = read("Name,Value\n\"unterminated", nil)
	require.Error(t, err)
	require.Nil(t, rows)

	_, err = read([]byte{0xff}, nil)
	require.ErrorContains(t, err, "not valid UTF-8")
}

func TestCsvReaderRowsDelimiterAndLegacyOutput(t *testing.T) {
	read, err := NewCsvReader(map[string]interface{}{"output": "rows", "delimiter": "｜"})
	require.NoError(t, err)
	rows, err := read("Name｜Value\nExample｜", nil)
	require.NoError(t, err)
	require.Equal(t, [][]string{{"Name", "Value"}, {"Example", ""}}, rows)

	_, err = NewCsvReader(map[string]interface{}{"output": "rows", "delimiter": "::"})
	require.ErrorContains(t, err, "invalid csv delimiter")

	legacyRead, err := NewCsvReader("name;value", ";")
	require.NoError(t, err)
	result, err := legacyRead("name;value\nExample;1\nOther;2", nil)
	require.NoError(t, err)
	require.Equal(t, "{\"name\":\"Example\",\"value\":\"1\"}\n{\"name\":\"Other\",\"value\":\"2\"}\n", result)
}

func TestCsvReaderRowsPreservesBlankRecords(t *testing.T) {
	read, err := NewCsvReader(map[string]interface{}{"output": "rows"})
	require.NoError(t, err)
	rows, err := read("\ufeff\r\nKey,Value\n\nA,\"first\r\n\r\nlast\"\r\n\r\n\"\"\n,\n\n", nil)
	require.NoError(t, err)
	require.Equal(t, [][]string{
		{}, {"Key", "Value"}, {}, {"A", "first\n\nlast"}, {}, {""}, {"", ""}, {},
	}, rows)
}
