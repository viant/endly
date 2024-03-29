package lambda

import (
	"crypto/sha256"
	"encoding/base64"
)

func hasDataChanged(data []byte, dataSha256 string) bool {
	algorithm := sha256.New()
	algorithm.Write(data)
	codeSha1 := algorithm.Sum(nil)
	dataSha1BAse64Encoded := base64.URLEncoding.EncodeToString(codeSha1)
	return dataSha256 != dataSha1BAse64Encoded
}
