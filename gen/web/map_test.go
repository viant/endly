package web

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/toolbox"
	"testing"
)

func TestMapSlice_Get(t *testing.T) {
	aMap := NewMap()
	aMap.Put("k1", 1)
	aMap.Put("k2", 2)
	aMap.Put("k3", NewMap())
	var k3 = aMap.GetMap("k3")
	k3.Put("k4", 4)
	assert.EqualValues(t, 2, aMap.Get("k2"))
	assert.EqualValues(t, 1, aMap.Get("k1"))
	aMap.Put("k2", 10)
	assert.EqualValues(t, 10, aMap.Get("k2"))
	assert.EqualValues(t, 4, aMap.GetMap("k3").Get("k4"))

	text, err := toolbox.AsYamlText(aMap)
	assert.Nil(t, err)
	assert.EqualValues(t, `k1: 1
k2: 10
k3:
  k4: 4
`, text)

}
