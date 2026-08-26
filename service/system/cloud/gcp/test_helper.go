package gcp

import (
	"log"

	"github.com/viant/endly/service/credential"
)

// HasTestCredentials returns true when e2e GCP credentials can be resolved via the alias registry.
func HasTestCredentials() bool {
	if credential.MappingConfigured() {
		return true
	}
	log.Print("skipping test")
	log.Print("configure e2e GCP credentials: op signin, then set E2E_CREDENTIALS_FILE or add resource/e2e-credentials.yaml (see doc/secrets/README.md)")
	return false
}
