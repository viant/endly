package gcp

import (
	"log"

	"github.com/viant/endly/service/credential"
)

// HasTestCredentials returns true when e2e GCP credentials are available via mapping or ~/.secret.
func HasTestCredentials() bool {
	if credential.MappingConfigured() || credential.LegacyE2ECredentialsConfigured() {
		return true
	}
	log.Print("skipping test")
	log.Print("configure e2e GCP credentials: ~/.secret/gcp-e2e.json, or op signin with resource/e2e-credentials.yaml (see doc/secrets/README.md)")
	return false
}
