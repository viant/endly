package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"testing"
)

func Test_IsAsciiPrintable(t *testing.T) {

	assert.False(t, endly.IsASCIIText(string([]byte{0x1, 0x3, 0x32})))
	assert.True(t, endly.IsASCIIText("abc\n"))
	assert.True(t, endly.IsASCIIText("abc\t"))
	assert.True(t, endly.IsASCIIText("abc\r"))
	assert.True(t, endly.IsASCIIText("<-abc->"))
	assert.True(t, endly.IsASCIIText("'"))
	assert.True(t, endly.IsASCIIText("\""))

}

func Test_AsPayload(t *testing.T) {

	{
		data := endly.AsPayload([]byte{0x1, 0x32})
		assert.Equal(t, "base64:ATI=", data)
	}
	{
		data := endly.AsPayload([]byte("abc\n\r "))
		assert.Equal(t, "abc\n\r ", data)
	}
}
