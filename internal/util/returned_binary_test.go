package util

import (
	"github.com/stretchr/testify/require"
	"github.com/viant/toolbox/data"
	"testing"
)

func TestReturnedBinaryRemainsLiteralBesideExpandedContainers(t *testing.T) {
	state := data.Map{"known": "expanded"}
	for _, binary := range [][]byte{nil, {}, []byte("${known} $unknown")} {
		require.False(t, IsContainer(binary))
		normalized, err := NormalizeValue(binary)
		require.NoError(t, err)
		require.Equal(t, binary, normalized)
		nested := map[string]interface{}{"bytes": binary, "template": "${known}", "empty": []interface{}{}}
		actual := ExpandReturnedContainer(state, nested).(map[string]interface{})
		require.Equal(t, binary, actual["bytes"])
		require.Equal(t, "expanded", actual["template"])
		require.Equal(t, []interface{}{}, actual["empty"])
	}
}
