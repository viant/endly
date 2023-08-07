package exec

import (
	"bytes"
	"github.com/viant/endly"
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

var defaultTarget *url.Resource

func SetDefaultTarget(context *endly.Context, target *url.Resource) {
	if target != nil {
		defaultTarget = target
	} else if target == nil {
		initDefaultTarget()
	}

	if target == nil {
		return
	}
	state := context.State()
	state.Put("execTarget", map[string]interface{}{
		"URL":         target.URL,
		"Credentials": target.Credentials,
	})
}

func initDefaultTarget() {
	privateKeyPath := path.Join(os.Getenv("HOME"), "/.secret/id_rsa")
	if !toolbox.FileExists(privateKeyPath) {
		privateKeyPath = path.Join(os.Getenv("HOME"), "/.ssh/id_rsa")
		if !toolbox.FileExists(privateKeyPath) {
			return
		}
	}
	defaultTarget = url.NewResource(defaultTargetURL)
	if defaultTarget.Credentials == "" {
		config := cred.Config{
			Username:       os.Getenv("USER"),
			PrivateKeyPath: privateKeyPath,
		}
		configData, _ := json.Marshal(config)
		if service, err := storage.NewServiceForURL(defaultTargetCredentialURL, ""); err == nil {
			if err = service.Upload(defaultTargetCredentialURL, bytes.NewReader(configData)); err == nil {
				defaultTarget.Credentials = defaultTargetCredentialURL
			}
		}
	}
}

// GetServiceTarget sets default target URL, credentials if emtpy
func GetServiceTarget(target *url.Resource) *url.Resource {
	if target != nil && target.Credentials != "" {
		return target
	}

	if defaultTarget == nil {
		initDefaultTarget()
	}
	return defaultTarget
}
