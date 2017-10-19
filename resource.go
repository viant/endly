package endly

import (
	"bytes"
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/cred"
	"github.com/viant/toolbox/storage"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

type Resource struct {
	Name        string
	Version     string
	URL         string
	Type        string
	Credential  string //name of file or alias to the file defined via credential service
	ParsedURL   *url.URL
	Cache       string
	CacheExpiry int
}

func (r *Resource) Clone() *Resource {
	return &Resource{
		Name:        r.Name,
		Version:     r.Version,
		URL:         r.URL,
		Type:        r.Type,
		Credential:  r.Credential,
		ParsedURL:   r.ParsedURL,
		Cache:       r.Cache,
		CacheExpiry: r.CacheExpiry,
	}
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
			return "", "", fmt.Errorf("Credential was empty: %v", r.Credential)
		}
		return "", "", nil
	}
	credential, err := cred.NewConfig(r.Credential)
	if err != nil {
		return "", "", reportError(fmt.Errorf("Failed to load Credential: %v %v", r.Credential, err))
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

//TODO support cache as dir
func (r *Resource) readFromCache() []byte {
	if toolbox.FileExists(r.Cache) {
		info, err := os.Stat(r.Cache)
		var isExpired = false
		if err == nil && r.CacheExpiry > 0 {
			elapsed := time.Now().Sub(info.ModTime())
			isExpired = elapsed > time.Second*time.Duration(r.CacheExpiry)
		}
		content, err := ioutil.ReadFile(r.Cache)
		if err == nil && !isExpired {
			return content
		}
	}
	return nil
}

func (r *Resource) Cachable() bool {
	return r.Cache != ""
}

func (r *Resource) Download() ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("Fail to download content on empty resource")
	}

	if r.Cachable() {
		content := r.readFromCache()
		if content != nil {
			return content, nil
		}
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
	if r.Cachable() {
		_ = ioutil.WriteFile(r.Cache, content, 0666)
	}
	return content, err
}

func NewResource(URL string) *Resource {
	URL = normalizeURL(URL)
	parsedURL, _ := url.Parse(URL)
	return &Resource{
		ParsedURL: parsedURL,
		URL:       URL,
	}
}

func normalizeURL(URL string) string {
	if strings.Contains(URL, "://") {
		return URL
	}
	if !strings.HasPrefix(URL, "/") {
		currentDirectory, err := os.Getwd()
		if err == nil {
			candidate := path.Join(currentDirectory, URL)
			if toolbox.FileExists(candidate) {
				URL = candidate
			}
		}
	}
	return toolbox.FileSchema + URL
}

const endlyRemoteRepo = "https://raw.githubusercontent.com/viant/endly/master/%v"

var endlyLocalRepo = fmt.Sprintf("file://%v/src/github.com/viant/endly/%v", os.Getenv("GOPATH"), "%v")

func NewEndlyRepoResource(context *Context, URI string) (*Resource, error) {
	var endlyLocalResource = fmt.Sprintf(endlyLocalRepo, URI)
	var localResource = NewResource(endlyLocalResource)
	var remoteResource = NewResource(fmt.Sprintf(endlyRemoteRepo, URI))
	if toolbox.FileExists(localResource.ParsedURL.Path) {
		return NewResource(endlyLocalResource), nil
	}
	_, err := context.Copy(false, remoteResource, localResource)
	if err != nil {
		return nil, err
	}
	return localResource, nil
}
