package storage

import (
	"errors"
	"io"
	"strings"
	"github.com/viant/afs/file"
	"github.com/viant/endly"
	"github.com/viant/toolbox/url"
	"os"
)

const defaultLineTemplate = "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua."

//CreateRequest represents a resources Upload request, it takes context state key to Upload to target destination.
type GenerateRequest struct {
	Template     string
	LineTemplate string
	Size         int
	SizeInMb     int
	Mode         int           `description:"os.FileMode"`
	Dest         *url.Resource `required:"true" description:"destination asset or directory"` //target URL with credentials
}

//CreateResponse represents a Upload response
type GenerateResponse struct {
	Size int
	URL  string
}

//Create creates a resource
func (s *service) Generate(context *endly.Context, request *GenerateRequest) (*GenerateResponse, error) {
	var response = &GenerateResponse{}
	err := s.generate(context, request, response)
	return response, err
}

func (s *service) generate(context *endly.Context, request *GenerateRequest, response *GenerateResponse) error {
	dest, storageOpts, err := GetResourceWithOptions(context, request.Dest)
	if err != nil {
		return err
	}
	fs, err := StorageService(context, dest)
	if err != nil {
		return err
	}
	response.URL = dest.URL
	reader := generateContent(context, request)
	return fs.Upload(context.Background(), dest.URL, os.FileMode(request.Mode), reader, storageOpts...)
}

func generateContent(context *endly.Context, request *GenerateRequest) io.Reader {
	if request.Template == "" {
		request.Template = " "
	}
	repeat := request.Size / len(request.Template)

	if ! strings.Contains(request.Template, "$") {
		text := strings.Repeat(request.Template, repeat+1)
		return strings.NewReader(string(text[:request.Size]))
	}

	state := context.State()
	state = state.Clone()
	items := make([]string, repeat + 1)

	for i := range items {
		state.Put("i", i)
		items[i] = state.ExpandAsText(request.Template)
	}
	text := strings.Join(items, "")
	return strings.NewReader(string(text[:request.Size]))
}

//Init initialises Upload request
func (r *GenerateRequest) Init() error {
	if r.Mode == 0 {
		r.Mode = int(file.DefaultFileOsMode)
	}
	if r.Template == "" && r.LineTemplate == "" {
		r.LineTemplate = defaultLineTemplate
	}
	if r.Template == "" {
		r.Template = r.LineTemplate + "\n"
	}
	if r.Size == 0 {
		r.Size = 1024 * 1024 * r.SizeInMb
	}
	return nil
}

//Validate checks if request is valid
func (r *GenerateRequest) Validate() error {
	if r.Dest == nil {
		return errors.New("dest was empty")
	}
	if r.Size == 0 && r.SizeInMb == 0 {
		return errors.New("size was empty")
	}

	return nil
}
