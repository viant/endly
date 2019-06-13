package exec

import (
	"bytes"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/storage"
	"github.com/viant/toolbox/url"
	"k8s.io/apimachinery/pkg/util/json"
	"os"
	"path"
)

const defaultTargetURL = "ssh://localhost/"
const defaultTargetCredentialURL = "mem:///localhost.json"

//SetDefaultTargetIfEmpty sets default target URL, credentials if emtpy
func SetDefaultTargetIfEmpty(target *url.Resource) *url.Resource {
	if target != nil && target.Credentials != "" {
		return target
	}
	privateKeyPath := path.Join(os.Getenv("HOME"), "/.secret/id_rsa")
	if !toolbox.FileExists(privateKeyPath) {
		privateKeyPath = path.Join(os.Getenv("HOME"), "/.ssh/id_rsa")
		if !toolbox.FileExists(privateKeyPath) {
			return target
		}
	}
	if target == nil {
		target = url.NewResource(defaultTargetURL)
	}
	if target.Credentials == "" {
		config := cred.Config{
			Username:       os.Getenv("USER"),
			PrivateKeyPath: privateKeyPath,
		}
		configData, _ := json.Marshal(config)
		if service, err := storage.NewServiceForURL(defaultTargetCredentialURL, ""); err == nil {
			if err = service.Upload(defaultTargetCredentialURL, bytes.NewReader(configData)); err == nil {
				target.Credentials = defaultTargetCredentialURL
			}
		}
	}
	return target
}
