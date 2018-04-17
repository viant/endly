package dsunit_test

import ("github.com/viant/toolbox"
"testing"
"github.com/viant/endly/testing/dsunit"
"github.com/viant/toolbox/data"
"strings"
"log"
"github.com/viant/neatly"
"github.com/stretchr/testify/assert"
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


func Test_DsUnitUdfGetTableRecords(t *testing.T) {
	var state = data.NewMap()


	collection := data.NewCollection()
	var referenceMap = data.NewMap()
	referenceMap.Put("id", 1000001)
	referenceMap.Put("name", "XZZZEE")
	collection.Push(referenceMap)
	state.SetValue("dsunit.AAAA.SSSSS", collection)
	state.Put("AsInt", neatly.AsInt)
	state.SetValue("uuid.next", 1)
	var records = []interface{}{}
	err := toolbox.NewJSONDecoderFactory().Create(strings.NewReader(JSON)).Decode(&records)
	if err != nil {
		log.Fatal(err)
	}
	state.Put("test", records)
	result, err:= dsunit.AsTableRecords("test", state)
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
