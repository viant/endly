package storage

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/viant/afs"
	afsfile "github.com/viant/afs/file"
	"github.com/viant/afs/matcher"
	"github.com/viant/afs/object"
	"github.com/viant/afs/option"
	afsstore "github.com/viant/afs/storage"
	"github.com/viant/endly/service/system/storage/copy"
)

// Dangerous provider regressions run only in this test binary's child process,
// with a hard deadline and a small stack. No MCP controller or real archive is used.
func TestStorageListRecursiveSyntheticZIP(t *testing.T) {
	for _, scenario := range []string{"explicit-directories", "content", "subtree", "implicit-directories"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestStorageListSyntheticZIPChild$", "-test.timeout=5s", "-test.v")
			command.Env = append(os.Environ(), "ENDLY_STORAGE_LIST_TEST_CHILD="+scenario)
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("synthetic ZIP child failed: %v (timeout=%v)\n%s", err, ctx.Err(), output)
			}
			if !strings.Contains(string(output), "synthetic ZIP scenario completed "+scenario) {
				t.Fatal("synthetic ZIP child did not execute its assertions")
			}
		})
	}
}

func TestStorageListSyntheticZIPChild(t *testing.T) {
	scenario := os.Getenv("ENDLY_STORAGE_LIST_TEST_CHILD")
	if scenario == "" {
		return
	}
	debug.SetMaxStack(4 << 20)
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	names := []string{"root.txt", "nested/", "nested/a.txt", "nested/deep/", "nested/deep/b.txt", "other/", "other/a.txt", "nested2/", "nested2/a.txt"}
	if scenario == "implicit-directories" {
		names = []string{"root.txt", "nested/a.txt", "nested/deep/b.txt", "other/a.txt", "nested2/a.txt"}
	}
	for _, name := range names {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(name, "/") {
			if _, err = file.Write([]byte("bytes:" + name)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(t.TempDir(), "synthetic.zip")
	if err := os.WriteFile(archive, buffer.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	root := "file:" + filepath.ToSlash(archive) + "/zip://localhost/"
	source := root
	request := &ListRequest{Recursive: true, Content: scenario == "content"}
	directory := false
	request.Match = &copy.Matcher{Basic: &matcher.Basic{Directory: &directory}}
	if scenario == "subtree" {
		source = root + "nested"
	}
	options, err := getMatcherOptions(request)
	if err != nil {
		t.Fatal(err)
	}
	response := &ListResponse{}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err = listResource(ctx, source, options, request, response); err != nil {
		t.Fatal(err)
	}
	expected := []string{"root.txt", "nested/a.txt", "nested/deep/b.txt", "other/a.txt", "nested2/a.txt"}
	if scenario == "subtree" {
		expected = []string{"nested/a.txt", "nested/deep/b.txt"}
	}
	actual := make([]string, 0, len(response.Assets))
	for _, resource := range response.Assets {
		member := strings.TrimPrefix(resource.Name, root)
		if member == resource.Name {
			t.Fatalf("member URL escaped archive root: %s", resource.Name)
		}
		actual = append(actual, member)
		if resource.Dir {
			t.Fatal("directory:false returned a directory")
		}
		if request.Content {
			if string(resource.Data) != "bytes:"+member {
				t.Fatalf("wrong content for member %s", member)
			}
		} else if len(resource.Data) != 0 {
			t.Fatal("metadata-only request read file content")
		}
	}
	sort.Strings(actual)
	sort.Strings(expected)
	if strings.Join(actual, "\n") != strings.Join(expected, "\n") {
		t.Fatalf("members differ: got %v want %v", actual, expected)
	}
	t.Log("synthetic ZIP scenario completed " + scenario)
}

type recursiveListFake struct {
	afs.Service
	objects      map[string][]afsstore.Object
	calls, opens int
	deep         bool
	cancel       context.CancelFunc
	walk         func(context.Context, string, afsstore.OnVisit, []afsstore.Option) error
}

func listTestObject(URL string, dir bool) afsstore.Object {
	mode := os.FileMode(0644)
	if dir {
		mode = os.ModeDir | 0755
	}
	return object.New(URL, afsfile.NewInfo(path.Base(URL), 0, mode, time.Time{}, dir), nil)
}
func (f *recursiveListFake) List(ctx context.Context, URL string, options ...afsstore.Option) ([]afsstore.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.calls++
	if f.cancel != nil {
		f.cancel()
	}
	entries := f.objects[listURLKey(URL)]
	if f.deep {
		entries = []afsstore.Object{listTestObject(URL, true), listTestObject(strings.TrimSuffix(URL, "/")+"/child", true)}
	}
	match, _ := option.GetListOptions(options)
	result := make([]afsstore.Object, 0, len(entries))
	for _, entry := range entries {
		if match(path.Dir(entry.URL()), entry) {
			result = append(result, entry)
		}
	}
	return result, nil
}
func (f *recursiveListFake) Open(ctx context.Context, entry afsstore.Object, options ...afsstore.Option) (io.ReadCloser, error) {
	f.opens++
	return io.NopCloser(strings.NewReader("content:" + entry.URL())), nil
}
func (f *recursiveListFake) Walk(ctx context.Context, URL string, visit afsstore.OnVisit, options ...afsstore.Option) error {
	return f.walk(ctx, URL, visit, options)
}

func TestStorageListRecursiveCycleAndFirstChild(t *testing.T) {
	const root = "mem://localhost/list"
	child := root + "/child"
	fake := &recursiveListFake{objects: map[string][]afsstore.Object{
		listURLKey(root):  {listTestObject(child, true), listTestObject(root+"/", true), listTestObject(root+"/root.txt", false)},
		listURLKey(child): {listTestObject(child+"/inner.txt", false), listTestObject(child+"/.", true), listTestObject(root, true)},
	}}
	request := &ListRequest{Recursive: true}
	response := &ListResponse{}
	if err := listResourceWithFS(context.Background(), fake, root, nil, request, response); err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, resource := range response.Assets {
		names = append(names, resource.Name)
	}
	sort.Strings(names)
	want := []string{child, child + "/inner.txt", root + "/root.txt"}
	sort.Strings(want)
	if strings.Join(names, "\n") != strings.Join(want, "\n") {
		t.Fatalf("first child/self identity lost: %v", names)
	}
	if fake.calls != 4 || fake.opens != 0 {
		t.Fatalf("cycle or metadata I/O: calls=%d opens=%d", fake.calls, fake.opens)
	}
	// A file-only output matcher must still visit the first real child directory.
	directory := false
	request.Match = &copy.Matcher{Basic: &matcher.Basic{Directory: &directory}}
	options, err := getMatcherOptions(request)
	if err != nil {
		t.Fatal(err)
	}
	response = &ListResponse{}
	if err = listResourceWithFS(context.Background(), fake, root, options, request, response); err != nil {
		t.Fatal(err)
	}
	if len(response.Assets) != 2 {
		t.Fatalf("directory:false pruned descendants: %v", response.Assets)
	}
	request.Content = true
	response = &ListResponse{}
	if err = listResourceWithFS(context.Background(), fake, root, options, request, response); err != nil {
		t.Fatal(err)
	}
	for _, resource := range response.Assets {
		if string(resource.Data) != "content:"+resource.Name {
			t.Fatal("ordinary content changed")
		}
	}
}

func TestStorageListRecursiveDepthAndCancellation(t *testing.T) {
	request := &ListRequest{Recursive: true}
	fake := &recursiveListFake{deep: true}
	if err := listResourceWithFS(context.Background(), fake, "mem://localhost/depth", nil, request, &ListResponse{}); err == nil || !strings.Contains(err.Error(), "maximum directory depth") {
		t.Fatalf("unbounded depth was not rejected: %v", err)
	}
	if fake.calls > 2*(maxRecursiveListDepth+1) {
		t.Fatalf("depth failure was unbounded: %d calls", fake.calls)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	fake = &recursiveListFake{}
	if err := listResourceWithFS(ctx, fake, "mem://localhost/cancel", nil, request, &ListResponse{}); !errors.Is(err, context.Canceled) || fake.calls != 0 {
		t.Fatalf("canceled context was replaced: %v calls=%d", err, fake.calls)
	}
	ctx, cancel = context.WithCancel(context.Background())
	fake = &recursiveListFake{cancel: cancel}
	if err := listResourceWithFS(ctx, fake, "mem://localhost/cancel", nil, request, &ListResponse{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("mid-list cancellation lost: %v", err)
	}
}

type listPanicReader struct{}

func (listPanicReader) Read([]byte) (int, error) { panic("Content:false must not consume a file body") }

type listErrorReader struct{ err error }

func (r listErrorReader) Read([]byte) (int, error) { return 0, r.err }

func TestStorageListArchiveMetadataAndFailure(t *testing.T) {
	const root = "file:/synthetic.zip/zip://localhost/"
	request := &ListRequest{Recursive: true}
	fake := &recursiveListFake{}
	fake.walk = func(ctx context.Context, URL string, visit afsstore.OnVisit, _ []afsstore.Option) error {
		_, err := visit(ctx, strings.TrimSuffix(root, "/"), "nested/", afsfile.NewInfo("a.txt", 1, 0644, time.Time{}, false), listPanicReader{})
		return err
	}
	response := &ListResponse{}
	if err := listResourceWithFS(context.Background(), fake, root, nil, request, response); err != nil {
		t.Fatal(err)
	}
	if len(response.Assets) != 1 || response.Assets[0].Name != root+"nested/a.txt" || len(response.Assets[0].Data) != 0 {
		t.Fatal("archive path or metadata-only semantics changed")
	}
	failure := errors.New("synthetic content failure")
	fake.walk = func(ctx context.Context, URL string, visit afsstore.OnVisit, _ []afsstore.Option) error {
		_, _ = visit(ctx, strings.TrimSuffix(root, "/"), "nested/", afsfile.NewInfo("a.txt", 1, 0644, time.Time{}, false), listErrorReader{failure})
		return nil // Model a provider replacing the callback error while closing.
	}
	request.Content = true
	if err := listResourceWithFS(context.Background(), fake, root, nil, request, &ListResponse{}); !errors.Is(err, failure) {
		t.Fatalf("archive callback error was lost: %v", err)
	}
	fake.walk = func(ctx context.Context, URL string, visit afsstore.OnVisit, _ []afsstore.Option) error {
		_, _ = visit(ctx, strings.TrimSuffix(root, "/"), "../", afsfile.NewInfo("escape.txt", 1, 0644, time.Time{}, false), listPanicReader{})
		return nil
	}
	request.Content = false
	ctx, cancel := context.WithCancel(context.Background())
	fake.walk = func(ctx context.Context, URL string, visit afsstore.OnVisit, _ []afsstore.Option) error {
		cancel()
		_, _ = visit(ctx, strings.TrimSuffix(root, "/"), "", afsfile.NewInfo("a.txt", 1, 0644, time.Time{}, false), listPanicReader{})
		return nil
	}
	if err := listResourceWithFS(ctx, fake, root, nil, request, &ListResponse{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("archive cancellation was lost: %v", err)
	}
	fake.walk = func(ctx context.Context, URL string, visit afsstore.OnVisit, _ []afsstore.Option) error {
		_, _ = visit(ctx, strings.TrimSuffix(root, "/"), "../", afsfile.NewInfo("escape.txt", 1, 0644, time.Time{}, false), listPanicReader{})
		return nil
	}
	request.Content = false
	if err := listResourceWithFS(context.Background(), fake, root, nil, request, &ListResponse{}); err == nil {
		t.Fatal("escaping archive member accepted")
	}
}
