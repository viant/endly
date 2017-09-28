package endly

import (
	"bytes"
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/storage"
	"io/ioutil"
	"net/url"
	"path"
	"strings"
)

type Resource struct {
	Name       string
	Version    string
	URL        string
	Type       string
	Credential string //name of file or alias to the file defined via credential service
	ParsedURL  *url.URL
}

func (r *Resource) Session() string {
	result := r.ParsedURL.Hostname() + ":" + r.ParsedURL.Port()
	if r.ParsedURL.User != nil {
		result = r.ParsedURL.User.Username() + "@" + result
	}
	return result
}

func (r *Resource) LoadCredential(errorIsEmpty bool) (string, string, error) {
	if r.Credential == "" {
		if errorIsEmpty {
			return "", "", fmt.Errorf("Credentail was empty: %v", r.Credential)
		}
		return "", "", nil
	}
	credential := &storage.PasswordCredential{}
	credentialResource := NewFileResource(r.Credential)
	err := credentialResource.JsonDecode(credential)
	if err != nil {
		return "", "", reportError(fmt.Errorf("Failed to load credentail: %v %v", r.Credential, err))
	}
	return credential.Username, credential.Password, nil
}

func (r *Resource) AuthURL() (string, error) {
	if r.Credential == "" {
		return r.URL, nil
	}
	username, password, err := r.LoadCredential(true)
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
	if r == nil {
		return reportError(fmt.Errorf("Fail to json decode on empty resource"))
	}
	var content, err = r.Download()
	if err != nil {
		return err
	}
	return toolbox.NewJSONDecoderFactory().Create(bytes.NewReader(content)).Decode(target)
}

func (r *Resource) Download() ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("Fail to download content on empty resource")
	}
	service, err := storage.NewServiceForURL(r.URL, r.Credential)
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

func NewResource(URL string) (*Resource, error) {
	parsedURL, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}
	return &Resource{
		ParsedURL: parsedURL,
		URL:       URL,
	}, nil
}

func NewFileResource(resource string) *Resource {
	if resource == "" {
		return nil
	}
	if !strings.HasPrefix(resource, "/") {
		fileName, _, _ := toolbox.CallerInfo(2)
		parent, _ := path.Split(fileName)
		resource = path.Join(parent, resource)
	}
	var URL = toolbox.FileSchema + resource
	parsedURL, _ := url.Parse(URL)
	return &Resource{
		ParsedURL: parsedURL,
		URL:       URL,
	}
}


const endlyRepo = "https://raw.githubusercontent.com/viant/endly/master/%v"

func NewEndlyRepoResource(context *Context, URI string) (*Resource, error) {
	localResource := NewFileResource(URI)
	if toolbox.FileExists(localResource.ParsedURL.Path) {
		return localResource, nil
	}
	remoteResource, err := NewResource(fmt.Sprintf(endlyRepo, URI))
	if err != nil {
		return nil, err
	}
	_, err = context.Copy(false, remoteResource, localResource)
	if err != nil {
		return nil, err
	}
	return localResource, nil
}
