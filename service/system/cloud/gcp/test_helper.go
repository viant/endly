package gcp

import (
	"log"
	"os"

	"github.com/viant/endly/service/credential"
	"github.com/viant/toolbox"
)

// HasTestCredentials returns true if e2e GCP credentials can be resolved.
func HasTestCredentials() bool {
	if credential.MappingConfigured() {
		return true
	}
	if os.Getenv("OP_INTEGRATION_REF") != "" {
		return true
	}
	secretPath := os.Getenv("HOME") + "/.secret/gcp-e2e.json"
	if toolbox.FileExists(secretPath) {
		return true
	}
	log.Print("skipping test")
	log.Print("configure e2e GCP credentials via E2E_CREDENTIALS_FILE, resource/e2e-credentials.yaml, or " + secretPath)
	return false
}
