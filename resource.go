package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/storage"
	"io/ioutil"
	"net/url"
	"path"
	"strings"
	"bytes"
)

type Resource struct {
	Name           string
	Version        string
	URL            string
	Type           string
	Credential     string
	CredentialFile string
	ParsedURL      *url.URL
}

func (r *Resource) Session() string {
	result := r.ParsedURL.Hostname() + ":" + r.ParsedURL.Port()
	if r.ParsedURL.User != nil {
		result = r.ParsedURL.User.Username() + "@" + result
	}
	return result
}

func (r *Resource) LoadCredential() (string, string, error) {
	if r.CredentialFile == "" {
		return "", "", nil
	}
	credential := &storage.PasswordCredential{}
	err := LoadCredential(r.CredentialFile, credential)
	if err != nil {
		return "", "", reportError(fmt.Errorf("Failed to load credentail: %v %v", r.CredentialFile, err))
	}
	return credential.Username, credential.Password, nil
}

func (r *Resource) AuthURL() (string, error) {
	if r.CredentialFile == "" {
		return r.URL, nil
	}
	username, password, err := r.LoadCredential()
	if err != nil {
		return "", err
	}
	return strings.Replace(r.URL, "//", "//"+username+"@"+password, 1), nil
}

func (r *Resource) DownloadText() (string, error) {
	var result, err = r.Download()
	if err != nil {
		return "", err
	}
	return string(result), err
}


func (r *Resource) JsonDecode(target interface{}) error {
	var content, err =  r.Download()
	if err != nil {
		return err
	}
	return toolbox.NewJSONDecoderFactory().Create(bytes.NewReader(content)).Decode(target)
}



func (r *Resource) Download() ([]byte, error) {
	service, err := storage.NewServiceForURL(r.URL, r.CredentialFile)
	if err != nil {
		return nil, err
	}
	object, err := service.StorageObject(r.URL)
	if err != nil {
		return nil, err
	}
	reader, err := service.Download(object)
	if err != nil {
		return nil, err
	}
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return content, err
}

func NewFileResource(resource string) *Resource {
	if !strings.HasPrefix(resource, "/") {
		fileName, _, _ := toolbox.CallerInfo(2)
		parent, _ := path.Split(fileName)
		resource = path.Join(parent, resource)
	}
	return &Resource{
		URL: toolbox.FileSchema + resource,
	}
}
