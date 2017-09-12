package endly

import (
	"fmt"
	"github.com/viant/toolbox"
)

func LoadCredential(credentailFile string, credential interface{}) error {
	if credential == "" {
		return nil
	}
	err := toolbox.LoadConfigFromUrl("file://"+credentailFile, credential)
	if err != nil {
		return reportError(fmt.Errorf("Failed to load credential %v: %v", credentailFile, err))
	}
	return nil
}
