package webdriver

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/viant/toolbox/data"
	"reflect"
	"testing"
)

func TestBinaryScreenshotSurvivesJSONTransport(t *testing.T) {
	png := []byte{0x89, 'P', 'N', 'G', 13, 10, 26, 10, 0xff, 0, 0x80}
	result := data.Map{}
	if !(&service{}).addResultIfPresent([]interface{}{png}, result, "screenshot") {
		t.Fatal("screenshot not captured")
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var received map[string]string
	if err = json.Unmarshal(encoded, &received); err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.StdEncoding.DecodeString(received["screenshot"])
	if err != nil || !bytes.Equal(decoded, png) {
		t.Fatalf("binary screenshot corrupted: %x %v", decoded, err)
	}
	png[0] = 0
	if result["screenshot"].([]byte)[0] != 0x89 {
		t.Fatal("result retained mutable driver buffer")
	}
}

func TestTextBytesAndWindowHandlesRemainUsable(t *testing.T) {
	s := &service{}
	result := data.Map{}
	s.addResultIfPresent([]interface{}{[]byte("legacy text")}, result, "text")
	if result["text"] != "legacy text" {
		t.Fatal("legacy text byte conversion changed")
	}
	handles := []string{"first", "second"}
	if !s.addResultIfPresent([]interface{}{handles}, result, "tabs") || !reflect.DeepEqual(result["tabs"], handles) {
		t.Fatal("window handles were dropped")
	}
}
