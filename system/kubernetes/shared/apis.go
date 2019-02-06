package shared

import "fmt"

var defaultKindAPIVersion = map[string]string{
	"CertificateSigningRequest": "certificates.k8s.io/v1beta1",
}

func LookupAPIVersion(kind string) (string, error) {
	result, ok := defaultKindAPIVersion[kind]
	if !ok {
		return "", fmt.Errorf("failed to lookup api: for %v", kind)
	}
	return result, nil
}
