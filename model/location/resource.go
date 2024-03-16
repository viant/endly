package location

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/viant/afs"
	"github.com/viant/afs/file"
	"github.com/viant/afs/option"
	"github.com/viant/afs/url"
	"github.com/viant/toolbox"
	"gopkg.in/yaml.v2"
	"path"
	"strings"
)


//Resource represents
type Resource struct {
	URL         string            `description:"resource URL or relative or absolute path" required:"true"` //URL of resource
	Credentials string            `description:"credentials file"`                                          //name of credential file or credential key depending on implementation
	CustomKey   *option.AES256Key `description:" content encryption key"`
	Key         string            `description:" secret key"`

}

//CredentialURL returns url's with provided credential
func (r *Resource) CredentialURL(username, password string) string {
	var urlCredential = ""
	if username != "" {
		urlCredential = username
		if password != "" {
			urlCredential += ":" + password
		}
		urlCredential += "@"
	}
	result := r.Scheme() + "://" + urlCredential + r.Host() + "/" + r.Path()
	return result
}


//Host return hostname[:port]
func (r *Resource) Host() string {
	ret := url.Host(r.URL)
	return ret
}

//Hostname return hostname
func (r *Resource) Hostname() string {
	ret := r.Host()
	if index := strings.LastIndex(ret, ":"); index != -1 {
		return ret[:index]
	}
	return ret
}

func (r *Resource) Path()  string {
	return url.Path(r.URL)
}

//Resource 
func (r *Resource) DecoderFactory() toolbox.DecoderFactory {
	ext := path.Ext(url.Path(r.URL))
	switch ext {
	case ".yaml", ".yml":
		return toolbox.NewFlexYamlDecoderFactory()
	default:
		return toolbox.NewJSONDecoderFactory()
	}
}

// JSONDecode decodes json resource into target
func (r *Resource) JSONDecode(ctx context.Context, fs afs.Service, target interface{}) error {
	return r.DecodeWith(ctx, fs, target, toolbox.NewJSONDecoderFactory())
}

// JSONDecode decodes yaml resource into target
func (r *Resource) YAMLDecode(ctx context.Context, fs afs.Service, target interface{}) error {
	if interfacePrt, ok := target.(*interface{}); ok {
		var data interface{}
		if err := r.DecodeWith(ctx, fs, &data, toolbox.NewYamlDecoderFactory()); err != nil {
			return err
		}
		if toolbox.IsSlice(data) {
			*interfacePrt = data
			return nil
		}
	}
	var mapSlice = yaml.MapSlice{}
	if err := r.DecodeWith(ctx, fs, &mapSlice, toolbox.NewYamlDecoderFactory()); err != nil {
		return err
	}
	if !toolbox.IsMap(target) {
		return toolbox.DefaultConverter.AssignConverted(target, mapSlice)
	}
	resultMap := toolbox.AsMap(target)
	for _, v := range mapSlice {
		resultMap[toolbox.AsString(v.Key)] = v.Value
	}
	return nil
}

//DownloadBase64 loads base64 resource content
func (r *Resource) DownloadBase64(ctx context.Context, fs afs.Service) (string, error) {
	data, err := fs.DownloadWithURL(ctx, r.URL)
	if err != nil {
		return "", err
	}
	_, err = base64.StdEncoding.DecodeString(string(data))
	if err == nil {
		return string(data), nil
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func (r *Resource) DecodeWith(ctx context.Context, fs afs.Service, target interface{}, decoderFactory toolbox.DecoderFactory) error {
	if r == nil {
		return fmt.Errorf("fail to %T decode on empty resource", decoderFactory)
	}
	if decoderFactory == nil {
		return fmt.Errorf("fail to decode %v, decoderFactory was empty", r.URL)
	}
	content, err := fs.DownloadWithURL(ctx, r.URL)
	if err != nil {
		return err
	}

	text := string(content)
	if toolbox.IsNewLineDelimitedJSON(text) {
		if aSlice, err := toolbox.NewLineDelimitedJSON(text); err == nil {
			return toolbox.DefaultConverter.AssignConverted(target, aSlice)
		}
	}
	err = decoderFactory.Create(bytes.NewReader(content)).Decode(target)
	if err != nil {
		return fmt.Errorf("failed to decode: %v, payload: %s", err, content)
	}
	return err
}



func (r *Resource) Port() string {
	host := r.Hostname()
	if index := strings.Index(host, ":"); index != -1 {
		return host[index+1:]
	}
	switch url.Scheme(r.URL, file.Scheme) {
	case "ssh", "scp":
		return "22"
	case "http":
		return "8080"
	}
	return ""
}

func (r *Resource) Scheme() string {
	return url.Scheme(r.URL, file.Scheme)
}

func (r *Resource) Decode(request interface{}) error {
	ctx := context.Background()
	fs := afs.New()
	ext := path.Ext(url.Path(r.URL))
	switch ext {
	case ".yaml", ".yml":
		return r.YAMLDecode(ctx, fs, request)
	}
	return r.DecodeWith(ctx, fs, request, r.DecoderFactory())
}

func (r *Resource) Rename(name string)  {
	var _, currentName = url.Split(r.URL, file.Scheme)
	if currentName == "" && strings.HasSuffix(r.URL, "/") {
		_, currentName = url.Split(r.URL[:len(r.URL)-1], file.Scheme)
		currentName += "/"
	}
	if index := strings.LastIndex(r.URL, currentName);index != -1 {
		r.URL = r.URL[:index] + name
	}
}

func (r *Resource) Clone() *Resource {
	ret  := *r
	return &ret
}

type Option func(o *Resource)

func WithCredentials(cred string) Option {
	return func(o *Resource) {
		o.Credentials = cred
	}
}

func NewResource(URL string, opts ...Option) *Resource {
	ret := &Resource{
		URL: URL,
	}
	for _, opt := range opts {
		opt(ret)
	}
	return ret
}
