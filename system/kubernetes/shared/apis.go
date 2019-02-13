package shared

import (
	"fmt"
	"strings"
)

var defaultKindAPIVersion = map[string][]string{
	"CertificateSigningRequest": {"certificates.k8s.io/v1beta1"},
}

func LookupAPIVersions(kind string) ([]string, error) {
	result, ok := defaultKindAPIVersion[kind]
	if !ok {
		return []string{}, fmt.Errorf("failed to lookup api: for %v", kind)
	}
	return result, nil
}

func LookupAPIVersion(kind string) (string, error) {
	versions, err := LookupAPIVersions(strings.Title(kind))
	if err != nil {
		return "", err
	}
	for _, ver := range versions {
		if ver == "v1" {
			return ver, nil
		}
	}
	return versions[0], nil

}
