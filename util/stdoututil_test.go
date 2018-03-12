package util_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly/util"
	"testing"
)

func Test_CheckNoSuchFileOrDirectory(t *testing.T) {
	assert.False(t, util.CheckNoSuchFileOrDirectory())
	assert.False(t, util.CheckNoSuchFileOrDirectory("abc"))
	assert.True(t, util.CheckNoSuchFileOrDirectory(" 1 no such file or directory "))
}

func Test_CheckCommandNotFound(t *testing.T) {
	assert.False(t, util.CheckCommandNotFound())
	assert.False(t, util.CheckCommandNotFound("abc "))
	assert.True(t, util.CheckCommandNotFound("command not found "))
	assert.True(t, util.CheckCommandNotFound("not installed "))
	assert.True(t, util.CheckCommandNotFound("Can't open "))
}

func Test_ExtractColumns(t *testing.T) {

	{
		var columns, ok = util.ExtractColumns("avc weww 33")
		assert.True(t, ok)
		assert.EqualValues(t, []string{"avc", "weww", "33"}, columns)
	}
	{
		_, ok := util.ExtractColumns("")
		assert.False(t, ok)
	}
}

func Test_ExtractColumn(t *testing.T) {

	{
		var column, ok = util.ExtractColumn("avc weww 33", 1)
		assert.True(t, ok)
		assert.EqualValues(t, "weww", column)
	}

	{
		_, ok := util.ExtractColumn("avc weww 33", 11)
		assert.False(t, ok)

	}

}
