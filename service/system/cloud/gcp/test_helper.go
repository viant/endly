package gcp

import (
	"log"
	"os"
	"path"

	"github.com/viant/toolbox"
)

// HasTestCredentials returns true when ~/.secret/gcp-e2e.json is present.
func HasTestCredentials() bool {
	secretPath := path.Join(os.Getenv("HOME"), ".secret", "gcp-e2e.json")
	if toolbox.FileExists(secretPath) {
		return true
	}
	log.Print("skipping test")
	log.Print("configure e2e GCP credentials: ~/.secret/gcp-e2e.json, or set credentialMap on the root workflow (see doc/secrets/README.md)")
	return false
}
