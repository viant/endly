package bigquery

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQueryRecordsPreservesScalarTypesAndNulls(t *testing.T) {
	input := `{"schema":{"fields":[{"name":"count","type":"INTEGER"},{"name":"ratio","type":"FLOAT64"},{"name":"enabled","type":"BOOL"},{"name":"name","type":"STRING"},{"name":"missing","type":"STRING"},{"name":"nullable","type":"STRING"}]},"rows":[{"f":[{"v":"0"},{"v":"1.25"},{"v":"false"},{"v":"a\"b"},{},{"v":null}]}]}`
	want := []map[string]interface{}{{"count": float64(0), "ratio": 1.25, "enabled": false, "name": "a\"b", "missing": nil, "nullable": nil}}
	for _, source := range []interface{}{input, []byte(input)} {
		actual, err := QueryRecords(source, nil)
		require.NoError(t, err)
		require.Equal(t, want, actual)
	}
	actual, err := QueryRecords(map[string]interface{}{"jobComplete": true}, nil)
	require.NoError(t, err)
	require.Equal(t, []map[string]interface{}{}, actual)
}

func TestQueryRecordsRejectsInvalidProtocolValues(t *testing.T) {
	for _, input := range []string{`{`, `{"schema":{"fields":[{"name":"n","type":"INT64"}]},"rows":[{"f":[{"v":"invalid"}]}]}`, `{"schema":{"fields":[{"name":"b","type":"BOOL"}]},"rows":[{"f":[{"v":"invalid"}]}]}`} {
		_, err := QueryRecords(input, nil)
		require.Error(t, err)
	}
}
