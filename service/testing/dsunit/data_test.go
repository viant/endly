package dsunit_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly/service/testing/dsunit"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/data"
	"log"
	"strings"
	"testing"
)

var JSON = `
[
   {
      "Table":"abc",
      "Value":[
         {
            "intBin":{
               "$AsInt(1)":"$AsInt(${dsunit.AAAA.SSSSS[0].id})",
               "$AsInt(2)":"$id"
            }
         }
      ],
      "AutoGenerate":{
         "id":"uuid.next"
      }
   }
]
`

var JSON2 = `
[
   {
      "Table":"xyz",
      "Value":[
         {
			"id": "1",
            "800": [
				"${featureAggData.featureAgg[0].ID}"
			]
         }
      ]
   }
]
`

func Test_DsUnitUdfGetTableRecords2(t *testing.T) {
	var state = data.NewMap()

	var records = []interface{}{}
	err := toolbox.NewJSONDecoderFactory().Create(strings.NewReader(JSON2)).Decode(&records)
	if err != nil {
		log.Fatal(err)
	}

	var collection = data.NewCollection()
	collection.Push(map[string]interface{}{
		"ID": 4,
	})
	collection.Push(map[string]interface{}{
		"ID": 2,
	})
	var featureAggData = data.NewMap()
	featureAggData.Put("featureAgg", collection)
	state.Put("featureAggData", featureAggData)
	state.Put("test", records)
	result, err := dsunit.AsTableRecords("test", state)
	assert.NotNil(t, result)
	tables := toolbox.AsMap(result)
	rows := tables["xyz"]
	row := toolbox.AsMap(toolbox.AsSlice(rows)[0])
	assert.NotNil(t, row)
	intBin, ok := row["800"].([]interface{})
	if assert.True(t, ok) {
		assert.EqualValues(t, 4, intBin[0])
	}
}

func Test_DsUnitUdfGetTableRecords(t *testing.T) {
	var state = data.NewMap()

	collection := data.NewCollection()
	var referenceMap = data.NewMap()
	referenceMap.Put("id", 1000001)
	referenceMap.Put("name", "XZZZEE")
	collection.Push(referenceMap)
	state.SetValue("dsunit.AAAA.SSSSS", collection)
	//state.Put("AsInt", neatly.AsInt)
	state.SetValue("uuid.next", 1)
	var records = []interface{}{}
	err := toolbox.NewJSONDecoderFactory().Create(strings.NewReader(JSON)).Decode(&records)
	if err != nil {
		log.Fatal(err)
	}
	state.Put("test", records)
	result, err := dsunit.AsTableRecords("test", state)
	tables := toolbox.AsMap(result)
	rows := tables["abc"]
	row := toolbox.AsMap(toolbox.AsSlice(rows)[0])
	assert.NotNil(t, row)
	intBin, ok := row["intBin"].(map[interface{}]interface{})
	if assert.True(t, ok) {
		assert.EqualValues(t, 1, intBin[2])
		assert.EqualValues(t, 1000001, intBin[1])
	}
}
