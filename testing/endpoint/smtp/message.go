package smtp

import (
	"io"
	"io/ioutil"
	"strings"
)

//Message represent an email
type Message struct {
	From    string
	To      []string
	Subject string
	Header  map[string]string
	Raw     string
	Body    string
}

func NewMessage(from string, to []string, reader io.Reader) (*Message, error) {
	result := &Message{
		From:   from,
		To:     to,
		Header: make(map[string]string),
	}
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	result.Raw = string(content)
	lines := strings.Split(result.Raw, "\n")
	for i, line := range lines {
		pair := strings.SplitN(line, ":", 2)
		if len(pair) != 2 {
			if i+1 < len(lines) {
				result.Body = strings.Join(lines[i+1:], "\n")
			}
			break
		}
		if pair[0] == "Subject" {
			result.Subject = strings.TrimSpace(pair[1])
		}
		result.Header[pair[0]] = pair[1]
	}
	return result, nil
}
