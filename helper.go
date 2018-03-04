package endly

import (
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"strings"
	"unicode"
)

//IsASCIIText return true if supplied string does not have binary data
func IsASCIIText(candidate string) bool {
	for _, r := range candidate {
		if r == '\n' || r == '\r' || r == '\t' {
			continue
		}
		if r > unicode.MaxASCII || !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

//FromPayload return bytes from
func FromPayload(payload string) ([]byte, error) {
	if strings.HasPrefix(payload, "text:") {
		return []byte(payload[5:]), nil
	} else if strings.HasPrefix(payload, "base64:") {
		payload = string(payload[7:])
		decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(payload))
		decoded, err := ioutil.ReadAll(decoder)
		if err != nil {
			return nil, err
		}
		return decoded, nil

	}
	return []byte(payload), nil
}

//AsPayload return string optionally encoded as base64 data has binary data.
func AsPayload(data []byte) string {
	if IsASCIIText(string(data)) {
		return string(data)
	}
	buf := new(bytes.Buffer)
	encoder := base64.NewEncoder(base64.StdEncoding, buf)
	encoder.Write(data)
	encoder.Close()
	return "base64:" + buf.String()
}
