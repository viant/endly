package storage

import (
	"errors"
	"fmt"
	"github.com/viant/afs/file"
	"github.com/viant/endly"
	"github.com/viant/endly/model/location"
	"github.com/viant/endly/model/msg"
	"io"
	"os"
	"strings"
	"sync"
)

const (
	defaultLineTemplate = "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua."
	fileNumberExpr      = "$fileNo"
)

// CreateRequest represents a resources Upload request, it takes context state key to Upload to target destination.
type GenerateRequest struct {
	Prefix        string `description:"Literal bytes written once before the template; included in Size"`
	Template      string
	LineTemplate  string
	Lines         int
	Size          int
	SizeInMb      int
	Index         int
	IndexVariable string
	Mode          int                `description:"os.FileMode"`
	Dest          *location.Resource `required:"true" description:"destination asset or directory"` //target URL with credentials
	FileCount     int
	InBackground  bool
}

// CreateResponse represents a Upload response
type GenerateResponse struct {
	Size int
	URLs []string
}

// Create creates a resource
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

	fileCount := request.FileCount
	if fileCount == 0 {
		fileCount = 1
	}
	URLs := []string{}
	readers := []io.Reader{}
	for i := 0; i < fileCount; i++ {
		reader := generateContent(context, request)
		readers = append(readers, reader)
		fileNumber := fmt.Sprintf("%04d", i)
		destURL := strings.Replace(dest.URL, fileNumberExpr, fileNumber, 1)
		URLs = append(URLs, destURL)
	}
	response.URLs = URLs
	waitGroup := &sync.WaitGroup{}
	for i := 0; i < fileCount; i++ {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			e := fs.Upload(context.Background(), URLs[index], os.FileMode(request.Mode), readers[index], storageOpts...)
			if e != nil {
				if request.InBackground {
					context.Publish(msg.NewErrorEvent(e.Error()))
				}
				err = e
			}
		}(i)
	}
	if request.InBackground {

		return err
	}
	waitGroup.Wait()
	return err
}

func generateContent(context *endly.Context, request *GenerateRequest) io.Reader {
	if request.Prefix == "" {
		return generateBody(context, request)
	}
	bodyRequest := *request
	bodyRequest.Prefix = ""
	if request.Lines == 0 && request.Size > 0 {
		bodyRequest.Size -= len(request.Prefix)
		if bodyRequest.Size <= 0 {
			return strings.NewReader(request.Prefix[:request.Size])
		}
	}
	body := generateBody(context, &bodyRequest)
	request.Index = bodyRequest.Index
	return io.MultiReader(strings.NewReader(request.Prefix), body)
}

// repeatedContent streams a non-empty literal without allocating the output.
type repeatedContent struct {
	pattern string
	offset  int
}

func (r *repeatedContent) Read(p []byte) (int, error) {
	n := copy(p, r.pattern[r.offset:])
	if n < len(p) {
		n += copy(p[n:], r.pattern[:r.offset])
	}
	for n < len(p) {
		n += copy(p[n:], p[:n])
	}
	r.offset = (r.offset + n) % len(r.pattern)
	return n, nil
}

func generateBody(context *endly.Context, request *GenerateRequest) io.Reader {
	if request.Template == "" {
		request.Template = " "
	}
	repeat := request.Lines
	if repeat == 0 {
		repeat = request.Size/len(request.Template) + 1
	}
	separator := ""
	if request.Lines > 0 {
		separator = "\n"
	}

	if !strings.Contains(request.Template, "$") {
		if request.Lines == 0 && request.Size > 0 {
			return io.LimitReader(&repeatedContent{pattern: request.Template}, int64(request.Size))
		}
		text := strings.Repeat(request.Template+separator, repeat)
		if request.Lines == 0 && request.Size > 0 {
			text = string(text[:request.Size])
		} else {
			text = strings.TrimSpace(text)
		}
		return strings.NewReader(text)
	}
	state := context.State()
	indexVariable := request.IndexVariable
	if indexVariable == "" {
		indexVariable = "i"
	}
	state = state.Clone()
	items := make([]string, repeat)
	for i := range items {
		state.Put(indexVariable, request.Index)
		request.Index++
		items[i] = state.ExpandAsText(request.Template)
		if request.Lines > 0 {
			items[i] = strings.TrimSpace(items[i])
		}
	}

	text := strings.Join(items, separator)

	if request.Lines == 0 && request.Size > 0 {
		text = string(text[:request.Size])
	}
	return strings.NewReader(text)
}

// Init initialises Upload request
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

// Validate checks if request is valid
func (r *GenerateRequest) Validate() error {
	if r.Size < 0 || r.SizeInMb < 0 || r.Lines < 0 || r.FileCount < 0 {
		return errors.New("size, line count and file count must be nonnegative")
	}
	if r.Lines == 0 && r.Size > 0 && len(r.Prefix) > r.Size {
		return errors.New("prefix exceeds requested size")
	}
	if r.Dest == nil {
		return errors.New("dest was empty")
	}
	if r.Size == 0 && r.SizeInMb == 0 && r.Lines == 0 {
		return errors.New("size was empty")
	}
	if r.FileCount > 1 && !strings.Contains(r.Dest.URL, fileNumberExpr) {
		return fmt.Errorf("dest.URL is missing %v variable for multi file generation, %v", fileNumberExpr, r.Dest.URL)
	}
	return nil
}
