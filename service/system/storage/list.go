package storage

import (
	"context"
	"github.com/pkg/errors"
	"github.com/viant/afs"
	"github.com/viant/afs/asset"
	"github.com/viant/afs/file"
	"github.com/viant/afs/matcher"
	"github.com/viant/afs/storage"
	afsurl "github.com/viant/afs/url"
	"github.com/viant/endly"
	"github.com/viant/endly/model/location"
	"github.com/viant/endly/service/system/storage/copy"
	"github.com/viant/endly/service/testing/validator"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

// DownloadRequest represents a resources Download request, it downloads source into context.state target key
type ListRequest struct {
	Source    *location.Resource `required:"true" description:"source asset or directory"`
	Match     *copy.Matcher
	Content   bool
	Recursive bool
	Expect    interface{}
}

// DownloadResponse represents a Download response
type ListResponse struct {
	URL    string
	Assets []*asset.Resource
	Assert *validator.AssertResponse
}

// Remove removes supplied assets
func (s *service) List(context *endly.Context, request *ListRequest) (*ListResponse, error) {
	var response = &ListResponse{
		Assets: make([]*asset.Resource, 0),
	}
	return response, s.list(context, request, response)
}

func (s *service) list(context *endly.Context, request *ListRequest, response *ListResponse) error {
	options, err := getMatcherOptions(request)
	if err != nil {
		return err
	}
	source, storageOpts, err := GetResourceWithOptions(context, request.Source, options...)
	if err != nil {
		return err
	}
	response.URL = source.URL
	if err = listResource(context.Background(), source.URL, storageOpts, request, response); err != nil {
		return err
	}
	if request.Expect != nil {
		response.Assert, err = validator.Assert(context, request, request.Expect, response.Assets, "List", "assert list responses")
	}
	return nil
}

// No provider depth option exists in the pinned AFS contract. Fail explicitly
// on unbounded virtual trees instead of overflowing the process stack.
const maxRecursiveListDepth = 1024

func listResource(ctx context.Context, URL string, storageOptions []storage.Option, request *ListRequest, response *ListResponse) error {
	return listResourceWithFS(ctx, fs, URL, storageOptions, request, response)
}

func listURLKey(URL string) string {
	base, location := afsurl.Base(afsurl.Normalize(URL, file.Scheme), file.Scheme)
	return base + "/" + strings.Trim(path.Clean("/"+location), "/")
}

func listResourceWithFS(ctx context.Context, source afs.Service, URL string, storageOptions []storage.Option, request *ListRequest, response *ListResponse) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Archive List results can lose member parents and repeat their own URL.
	// The native provider walker preserves parent/member identity without descent.
	if request.Recursive && afsurl.SchemeExtensionURL(URL) != "" {
		return listArchiveResources(ctx, source, URL, storageOptions, request, response)
	}
	type pendingDirectory struct {
		URL   string
		depth int
	}
	pending := []pendingDirectory{{URL: URL}}
	visited := map[string]bool{}
	emitted := map[string]bool{}
	for len(pending) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		next := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		key := listURLKey(next.URL)
		if visited[key] {
			continue
		}
		if next.depth > maxRecursiveListDepth {
			return errors.New("storage:list recursive listing exceeded maximum directory depth")
		}
		visited[key] = true
		objects, err := source.List(ctx, next.URL, storageOptions...)
		if err != nil {
			return err
		}
		for _, object := range objects {
			if err := ctx.Err(); err != nil {
				return err
			}
			objectKey := listURLKey(object.URL())
			if object.IsDir() && (objectKey == key || visited[objectKey]) || emitted[objectKey] {
				continue
			}
			var resource *asset.Resource
			if object.IsDir() {
				resource = asset.NewDir(object.URL(), object.Mode())
			} else {
				resource = asset.NewFile(object.URL(), nil, object.Mode())
			}
			if request.Content && !object.IsDir() {
				reader, err := source.Open(ctx, object)
				if err != nil {
					return errors.Wrap(err, "failed to download listed content")
				}
				resource.Data, err = ioutil.ReadAll(reader)
				_ = reader.Close()
				if err != nil {
					return errors.Wrap(err, "failed to read listed content")
				}
			}
			response.Assets = append(response.Assets, resource)
			emitted[objectKey] = true
		}
		if !request.Recursive {
			continue
		}
		dirMatcher := &matcher.Basic{Directory: &request.Recursive}
		// Copy options: appending a traversal matcher must not replace the caller's
		// output matcher through shared backing storage. Directory:false filters
		// returned assets, never whether a child directory is visited.
		recursiveOptions := append(append([]storage.Option(nil), storageOptions...), dirMatcher.Match)
		directories, err := source.List(ctx, next.URL, recursiveOptions...)
		if err != nil {
			return err
		}
		for i := len(directories) - 1; i >= 0; i-- {
			object := directories[i]
			if !object.IsDir() || listURLKey(object.URL()) == key || visited[listURLKey(object.URL())] {
				continue
			}
			pending = append(pending, pendingDirectory{URL: object.URL(), depth: next.depth + 1})
		}
	}
	return ctx.Err()
}

func listArchiveResources(ctx context.Context, source afs.Service, URL string, options []storage.Option, request *ListRequest, response *ListResponse) error {
	root := strings.Trim(path.Clean("/"+afsurl.Path(URL)), "/")
	emitted := map[string]bool{}
	var visitErr error
	err := source.Walk(ctx, URL, func(_ context.Context, base, parent string, info os.FileInfo, reader io.Reader) (bool, error) {
		fail := func(err error) (bool, error) { visitErr = err; return false, err }
		if err := ctx.Err(); err != nil {
			return fail(err)
		}
		raw := parent + info.Name()
		if path.IsAbs(raw) {
			return fail(errors.New("storage:list archive member path is absolute"))
		}
		for _, part := range strings.Split(raw, "/") {
			if part == ".." {
				return fail(errors.New("storage:list archive member escapes its source path"))
			}
		}
		member := strings.Trim(path.Clean("/"+raw), "/")
		if root != "" && member != root && !strings.HasPrefix(member, root+"/") {
			return true, nil
		}
		if member == root && info.IsDir() {
			return true, nil
		}
		relative := strings.TrimPrefix(strings.TrimPrefix(member, root), "/")
		depth := strings.Count(relative, "/")
		if info.IsDir() && relative != "" {
			depth++
		}
		if depth > maxRecursiveListDepth {
			return fail(errors.New("storage:list recursive listing exceeded maximum directory depth"))
		}
		memberURL := afsurl.Join(base, member)
		key := listURLKey(memberURL)
		if emitted[key] {
			if info.IsDir() {
				return true, nil
			}
			return fail(errors.New("storage:list archive contains a duplicate member path"))
		}
		var resource *asset.Resource
		if info.IsDir() {
			resource = asset.NewDir(memberURL, info.Mode())
		} else {
			resource = asset.NewFile(memberURL, nil, info.Mode())
		}
		if request.Content && !info.IsDir() {
			if reader == nil {
				return fail(errors.New("storage:list archive content reader is unavailable"))
			}
			resource.Data, visitErr = ioutil.ReadAll(reader)
			if visitErr != nil {
				return false, visitErr
			}
		}
		emitted[key] = true
		response.Assets = append(response.Assets, resource)
		return true, nil
	}, options...)
	// Some archive walkers replace callback errors while closing their reader.
	// Retain the original failure and never turn an incomplete walk into success.
	if visitErr != nil {
		return visitErr
	}
	if err != nil {
		return err
	}
	return ctx.Err()
}

func getMatcherOptions(request *ListRequest) ([]storage.Option, error) {
	var options = make([]storage.Option, 0)
	if request.Match != nil {
		matcher, err := request.Match.Matcher()
		if err != nil {
			return nil, err
		}
		options = append(options, matcher)
	}
	return options, nil
}

// Validate checks if request is valid
func (r *ListRequest) Validate() error {
	if r.Source == nil {
		return errors.New("source was empty")
	}
	return nil
}
