package common

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/toolbox"
	"testing"
)

func TestMap_GetValue(t *testing.T) {

	aMap := NewMap()

	{
		metaMap := make(map[string]int)
		metaMap["USER"] =7
		aMap.Put("meta", metaMap)

		value, ok := aMap.GetValue("meta.USER")
		assert.True(t, ok)
		if ! assert.Equal(t, 7, value) {
			return
		}
		aMap.SetValue("meta.USER", toolbox.AsInt(value)+1)
		value, ok = aMap.GetValue("meta.USER")
		assert.True(t, ok)
		if ! assert.Equal(t, 8, value) {
			return
		}

	}

	{
		var collection = NewCollection()
		collection.Push("1")
		collection.Push("20")
		collection.Push("30")
		aMap.Put("collection", collection)

		var subMap = NewMap()
		subMap.Put("i", 10)
		subMap.Put("col", collection)
		aMap.Put("a", subMap)
		aMap.Put("b", "123")
		aMap.Put("c", "b")
	}

	{ //test simple get operation
		value, has := aMap.GetValue("c")
		assert.True(t, has)
		assert.Equal(t, "b", value)

	}

	{ //test  get operation
		value, has := aMap.GetValue("a.col")
		assert.True(t, has)
		assert.Equal(t, []interface{}{"1", "20", "30"}, toolbox.AsSlice(value))

	}
	{ //test reference get operation
		value, has := aMap.GetValue("$c")
		assert.True(t, has)
		assert.Equal(t, "123", value)

	}

	{ //test post increment operations
		value, has := aMap.GetValue("a.i++")
		assert.True(t, has)
		assert.Equal(t, 10, value)
		value, has = aMap.GetValue("a.i++")
		assert.True(t, has)
		assert.Equal(t, 11, value)
	}

	{ //test pre increment operations
		value, has := aMap.GetValue("++a.i")
		assert.True(t, has)
		assert.Equal(t, 13, value)
		value, has = aMap.GetValue("++a.i")
		assert.True(t, has)
		assert.Equal(t, 14, value)
	}

	{ //	test shift
		value, has := aMap.GetValue("<-collection")
		assert.True(t, has)
		assert.Equal(t, "1", value)
		value, has = aMap.GetValue("<-collection")
		assert.True(t, has)
		assert.Equal(t, "20", value)

	}


}

func TestMap_SetValue(t *testing.T) {

	aMap := NewMap()

	{ // test simple Set

		_, has := aMap.GetValue("z.a")
		assert.False(t, has)
		aMap.SetValue("z.a", "123")
		value, has := aMap.GetValue("z.a")
		assert.True(t, has)
		assert.Equal(t, "123", value)
	}

	{ // test reference set

		aMap.SetValue("z.b", "111")
		value, has := aMap.GetValue("z.b")
		assert.True(t, has)
		assert.Equal(t, "111", value)

		aMap.SetValue("zzz", "z.b")
		aMap.SetValue("$zzz", "222")
		value, has = aMap.GetValue("z.b")
		assert.True(t, has)
		assert.Equal(t, "222", value)
	}

	{
		//test push
		aMap.SetValue("->a.v", 1)
		aMap.SetValue("->a.v", 2)

		aCollection, has := aMap.GetValue("a.v")
		assert.True(t, has)
		assert.Equal(t, []interface{}{1, 2}, toolbox.AsSlice(aCollection))
	}

}
