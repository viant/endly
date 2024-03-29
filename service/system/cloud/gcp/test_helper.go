package gcp

import (
	"github.com/viant/toolbox"
	"log"
	"os"
	"path"
)

// HasTestCredentialSetup returns true if e2e test credentials are set
func HasTestCredentials() bool {
	secretPath := path.Join(os.Getenv("HOME"), ".secret/gcp-e2e.json")
	if toolbox.FileExists(path.Join(os.Getenv("HOME"), ".secret/gcp-e2e.json")) {
		return true
	}
	log.Print("skipping test")
	log.Print("Create e2e dedicated project service account credentials and store it in " + secretPath)
	return false
}
