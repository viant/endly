package migrator

import (
	"encoding/json"
	"fmt"
	"strings"
)

//build to build the objects that will serialize into the run.yaml, send.yaml, request.json and environment json files
const RunYaml = `
init:
  {{INIT_PART}}
pipeline:
  info:
    action: print
    message: $environment
  runRequests:
    tag: $pathMatch
    data:
      '${tagId}.[]requests': '@request.json'
    subPath: requests/${index}_*
    range: 1..{{REQUEST_LEN}}
    template:
      run:
        init:
          tagId: $tagId
        action: run
        request: '@send'
`

const SendYaml = `
init:
  req: ${data.${tagId}.requests}
pipeline:
  send:
    action: 'http/runner:send'
    requests: $req
  test:
    workflow: assert
    actual: 
      Code: $send.Responses[0].Code
    expected:
      Code: 404
`

type cookieBuilder struct {
	Domain     string
	Expires    string
	HttpOnly   bool
	MaxAge     int
	Name       string
	Path       string
	Raw        string
	RawExpires string
	SameSite   int
	Secure     bool
	Unparsed   []string
	Value      string
}

type requestBuilder struct {
	Body    string              `json:",omitempty"`
	Cookies []*cookieBuilder    `json:",omitempty"`
	Header  map[string][]string `json:",omitempty"`
	Method  string
	URL     string
	name    string
}

func (b *requestBuilder) addCookie() *cookieBuilder {
	c := &cookieBuilder{}
	b.Cookies = append(b.Cookies, c)
	return c
}

func (b *requestBuilder) addHeader(key string, value []string) {
	b.Header[key] = value
}

func (b *requestBuilder) TOJson() (string, error) {
	s, err := json.MarshalIndent(b, "", "    ")
	if err != nil {
		return "", err
	}

	return string(s), nil
}

type environmentBuilder struct {
	Variables map[string]string
	name      string
}

func (b *environmentBuilder) addVariable(key string, value string) {
	b.Variables[key] = value
}

func (b *environmentBuilder) TOJson() (string, error) {
	s, err := json.MarshalIndent(b.Variables, "", "    ")
	if err != nil {
		return "", err
	}

	return string(s), nil
}

type runBuilder struct {
	variables    map[string]interface{}
	requests     []*requestBuilder
	environments []*environmentBuilder
}

func (b *runBuilder) addEnvironment(name string) *environmentBuilder {
	e := &environmentBuilder{
		Variables: make(map[string]string),
		name:      name,
	}
	b.environments = append(b.environments, e)
	return e
}

func (b *runBuilder) addRequest(method string, uRL string, name string) *requestBuilder {
	r := &requestBuilder{
		Method: method,
		URL:    uRL,
		name:   name,
		Header: make(map[string][]string),
	}
	b.requests = append(b.requests, r)
	return r
}

func (b *runBuilder) addVariable(key string, value interface{}) {
	b.variables[key] = value
}

func (b *runBuilder) sendToYaml() string {
	return SendYaml
}

func (b *runBuilder) toYamlInitPart() string {
	var sb strings.Builder

	if len(b.environments) > 0 {
		sb.WriteString("environment: ")
		sb.WriteString("$AsData($Cat('")
		sb.WriteString(makeDirOrFileName(b.environments[0].name))
		sb.WriteString(".json'))\n")

		for k := range b.environments[0].Variables {
			sb.WriteString("  ")
			sb.WriteString(k)
			sb.WriteString(": $environment.")
			sb.WriteString(k)
			sb.WriteString("\n")
		}
	}

	for k, v := range b.variables {
		sb.WriteString("  ")
		sb.WriteString(k)
		sb.WriteString(": ")
		sb.WriteString(v.(string))
		sb.WriteString("\n")
	}

	return sb.String()
}

func (b *runBuilder) runToYaml() string {
	l := strings.TrimLeft(fmt.Sprintf("%03s", fmt.Sprint(len(b.requests))), " ")
	return strings.ReplaceAll(strings.ReplaceAll(RunYaml, "{{INIT_PART}}", b.toYamlInitPart()), "{{REQUEST_LEN}}", l)
}

func NewRunBuilder() *runBuilder {
	return &runBuilder{
		variables: make(map[string]interface{}),
	}
}
