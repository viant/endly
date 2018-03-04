package util_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"testing"
)

func Test_CheckNoSuchFileOrDirectory(t *testing.T) {
	assert.False(t, endly.CheckNoSuchFileOrDirectory())
	assert.False(t, endly.CheckNoSuchFileOrDirectory("abc"))
	assert.True(t, endly.CheckNoSuchFileOrDirectory(" 1 no such file or directory "))
}

func Test_CheckCommandNotFound(t *testing.T) {
	assert.False(t, endly.CheckCommandNotFound())
	assert.False(t, endly.CheckCommandNotFound("abc "))
	assert.True(t, endly.CheckCommandNotFound("command not found "))
	assert.True(t, endly.CheckCommandNotFound("not installed "))
	assert.True(t, endly.CheckCommandNotFound("Can't open "))
}

func Test_ExtractColumns(t *testing.T) {

	{
		var columns, ok = endly.ExtractColumns("avc weww 33")
		assert.True(t, ok)
		assert.EqualValues(t, []string{"avc", "weww", "33"}, columns)
	}
	{
		_, ok := endly.ExtractColumns("")
		assert.False(t, ok)
	}
}

func Test_ExtractColumn(t *testing.T) {

	{
		var column, ok = endly.ExtractColumn("avc weww 33", 1)
		assert.True(t, ok)
		assert.EqualValues(t, "weww", column)
	}

	{
		_, ok := endly.ExtractColumn("avc weww 33", 11)
		assert.False(t, ok)

	}

}
