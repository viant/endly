package util

import (
	"bytes"
	"encoding/base64"
	"github.com/viant/toolbox"
	"io/ioutil"
	"strings"
)

// FromPayload return bytes from
func FromPayload(payload string) ([]byte, error) {
	if strings.HasPrefix(payload, "text:") {
		return []byte(payload[5:]), nil
	} else if strings.HasPrefix(payload, "base64:") {
		payload = string(payload[7:])
		if payload == "" {
			return []byte{}, nil
		}
		decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(payload))
		decoded, err := ioutil.ReadAll(decoder)
		if err != nil {
			return nil, err
		}
		return decoded, nil

	}
	return []byte(payload), nil
}

// AsPayload return string optionally encoded as base64 data has binary data.
func AsPayload(data []byte) string {
	if toolbox.IsASCIIText(string(data)) {
		return string(data)
	}
	buf := new(bytes.Buffer)
	encoder := base64.NewEncoder(base64.StdEncoding, buf)
	encoder.Write(data)
	encoder.Close()
	return "base64:" + buf.String()
}
