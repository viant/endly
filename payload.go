package endly

import (
	"strings"
	"encoding/base64"
	"io/ioutil"
	"unicode"
)




//IsAsciiPrintable check supplied string does not have binary data
func IsAsciiPrintable(s string) bool {
	for _, r := range s {
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
