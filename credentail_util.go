package endly

import (
	"fmt"
	"github.com/viant/toolbox"
)

func LoadCredential(credentialFile string, credential interface{}) error {
	if credential == "" {
		return nil
	}
	err := toolbox.LoadConfigFromUrl("file://"+credentialFile, credential)
	if err != nil {
		return reportError(fmt.Errorf("Failed to load credential %v: %v", credentialFile, err))
	}
	return nil
}
