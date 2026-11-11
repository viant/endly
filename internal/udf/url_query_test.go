package udf

import (
	"encoding/json"
	"errors"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/viant/endly"
	"github.com/viant/toolbox/data"
	toolboxudf "github.com/viant/toolbox/data/udf"
)

func TestURLSetQuery(t *testing.T) {
	payload := `{"id":52,"zero":0,"flag":false,"null":null,"text":"雪 &+\"'%#"}`
	for _, source := range []string{
		"https://example.test/report?keep=one&parameters=old&keep=two&parameters=stale&note=a%26b%2Bc#section%202",
		"https://example.test/report?keep=one&keep=two&note=a%26b%2Bc#section%202",
	} {
		actual, err := URLSetQuery([]interface{}{source, "parameters", payload}, nil)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := url.Parse(actual.(string))
		if err != nil {
			t.Fatal(err)
		}
		query := parsed.Query()
		if !reflect.DeepEqual(query["keep"], []string{"one", "two"}) || query.Get("note") != "a&b+c" || parsed.Fragment != "section 2" {
			t.Fatalf("unrelated URL meaning changed: %s", actual)
		}
		if !reflect.DeepEqual(query["parameters"], []string{payload}) {
			t.Fatalf("replacement changed: %q", query["parameters"])
		}
	}
	actual, err := URLSetQuery([]interface{}{"https://example.test/report#fragment", "space &+雪", ""}, nil)
	if err != nil {
		t.Fatal(err)
	}
	parsed, _ := url.Parse(actual.(string))
	if !reflect.DeepEqual(parsed.Query()["space &+雪"], []string{""}) || parsed.Fragment != "fragment" {
		t.Fatalf("empty value lost: %s", actual)
	}
}

func TestURLSetQueryErrors(t *testing.T) {
	for _, input := range []interface{}{nil, "url", []interface{}{}, []interface{}{"url", "key"}, []interface{}{"url", "key", 52}} {
		actual, err := URLSetQuery(input, nil)
		if err == nil || actual != nil {
			t.Fatalf("invalid shape accepted: %#v", input)
		}
	}
	for _, source := range []string{"https://example.test/%zz?token=private-marker", "https://example.test/?token=private-marker&bad=%zz", "https://example.test/?token=private-marker&bad=a;b"} {
		actual, err := URLSetQuery([]interface{}{source, "key", "value"}, nil)
		if err == nil || actual != nil {
			t.Fatal("malformed URL/query returned partial success")
		}
		if strings.Contains(err.Error(), "private-marker") || strings.Contains(err.Error(), source) {
			t.Fatalf("input leaked: %v", err)
		}
		if strings.Contains(source, "%zz") {
			var escape url.EscapeError
			if !errors.As(err, &escape) {
				t.Fatalf("escape cause lost: %v", err)
			}
		}
	}
}

func TestURLSetQueryNativeExpansion(t *testing.T) {
	state := data.NewMap()
	toolboxudf.Register(state)
	functions := data.NewMap()
	for name, function := range state[data.UDFKey].(data.Map) {
		functions[name] = function
	}
	functions["URLSetQuery"] = endly.PredefinedUdfs["URLSetQuery"]
	state[data.UDFKey] = functions
	state["sourceURL"] = "https://example.test/?parameters=old&parameters=duplicate#section"
	state["parameters"] = map[string]interface{}{"id": 52, "trial": false, "empty": []interface{}{}, "nullable": nil, "text": "雪 &+\"'%#"}
	state["parametersJSON"] = state.Expand("$AsJSON('$parameters')")
	actual := state.Expand("$URLSetQuery($sourceURL, 'parameters', $parametersJSON)")
	parsed, err := url.Parse(actual.(string))
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(parsed.Query().Get("parameters")), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["id"] != float64(52) || decoded["trial"] != false || decoded["nullable"] != nil || decoded["text"] != state["parameters"].(map[string]interface{})["text"] || len(parsed.Query()["parameters"]) != 1 {
		t.Fatalf("typed parameter contract changed: %#v", decoded)
	}
}
