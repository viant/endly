package xunit

import (
	"bytes"
	"encoding/xml"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)

func Test_Encode(t *testing.T) {

	var testsuites = NewTestsuite()
	testsuites.Name = "abc"
	buf := new(bytes.Buffer)
	err := xml.NewEncoder(buf).EncodeElement(testsuites, xml.StartElement{Name: xml.Name{Local: "test-suite"}})
	if err != nil {
		log.Fatal(err)
	}
	assert.True(t, buf.String() != "")

}
