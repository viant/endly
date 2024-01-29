package migrator

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"regexp"
	"strings"
)

const VarScopeNode = "_postman_variable_scope"
const ScopeValueEnv = "environment"
const ScopeValueGlobal = "globals"
const InfoNode = "info"
const PostmanIdNode = "_postman_id"

//*** logic helpers are cleaner to unit test so separate from file helpers ***

func parsePostmanReader(r io.Reader) (*postmanObject, error) {
	var nodes map[string]interface{}
	var pt postmanType = notPostman

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &nodes)
	if err != nil {
		return nil, err
	}

	val, ok := nodes[VarScopeNode]
	if ok {
		if val == ScopeValueEnv {
			pt = environment
		} else if val == ScopeValueGlobal {
			pt = globals
		}
	}

	_, ok = nodes[InfoNode]
	if ok {
		_, ok = nodes[InfoNode].(map[string]interface{})[PostmanIdNode]
		if ok {
			pt = requests
		}
	} else if pt == notPostman {
		return &postmanObject{
			nodes:      nil,
			objectType: pt,
		}, nil
	}

	return &postmanObject{
		nodes:      nodes,
		objectType: pt,
	}, nil
}

func convertToRunBuilder(objects []*postmanObject) *runBuilder {
	b := NewRunBuilder()
	for _, o := range objects {
		switch o.objectType {
		case environment:
			e := b.addEnvironment(o.nodes["name"].(string))
			for _, v := range o.nodes["values"].([]interface{}) {
				m := v.(map[string]interface{})
				e.addVariable(m["key"].(string), m["value"].(string))
			}
		case globals:
			for _, v := range o.nodes["values"].([]interface{}) {
				m := v.(map[string]interface{})
				b.addVariable(m["key"].(string), m["value"].(string))
			}
		case requests:
			for _, v := range o.nodes["variable"].([]interface{}) {
				v := v.(map[string]interface{})
				b.addVariable(v["key"].(string), v["value"].(string))
			}

			for _, v := range o.nodes["item"].([]interface{}) {
				v := v.(map[string]interface{})
				name := v["name"].(string)
				m := v["request"].(map[string]interface{})
				url := m["url"].(map[string]interface{})["raw"].(string)
				r := b.addRequest(m["method"].(string), replaceEndlyVar(url), name)

				if h, ok := m["header"]; ok {
					h := h.([]interface{})
					for _, i := range h {
						i := i.(map[string]interface{})
						key := i["key"].(string)
						value := replaceEndlyVar(i["value"].(string))
						r.addHeader(key, []string{value})
					}
				}

				if b, ok := m["body"]; ok {
					b := b.(map[string]interface{})
					r.Body = replaceEndlyVar(b["raw"].(string))
				}
			}
		}
	}

	return b
}

func makeDirOrFileName(str string) string {
	str = strings.ReplaceAll(str, "_", " ")
	return strings.ReplaceAll(regexp.MustCompile(`[^a-zA-Z0-9 ]+`).ReplaceAllString(str, ""), " ", "_")
}

func replaceEndlyVar(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "{{", "${"), "}}", "}")
}
