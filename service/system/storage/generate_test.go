package storage

import (
	"io"
	"strings"
	"testing"

	"github.com/viant/endly/model/location"
)

func TestGenerateLiteralContent(t *testing.T) {
	for _, tt := range []struct {
		name, pattern, prefix string
		size, lines           int
		want                  string
	}{
		{"partial repetition", "abc", "", 8, 0, "abcabcab"},
		{"prefix included in size", "abc", "header\n", 12, 0, "header\nabcab"},
		{"prefix only", "abc", "head", 4, 0, "head"},
		{"legacy lines", " a ", "", 0, 2, "a \n a"},
		{"prefix lines", "a", "head\n", 0, 2, "head\na\na"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			request := &GenerateRequest{Template: tt.pattern, Prefix: tt.prefix, Size: tt.size, Lines: tt.lines}
			got, err := io.ReadAll(generateContent(nil, request))
			if err != nil || string(got) != tt.want {
				t.Fatalf("got %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}

func TestGenerateLargeLiteralStreams(t *testing.T) {
	// Consume only a small prefix: creating a multi-gigabyte fixture must not
	// materialize its declared size before the storage service starts reading.
	r := generateContent(nil, &GenerateRequest{Template: "abcdef", Size: 4_096_000_000})
	got, err := io.ReadAll(io.LimitReader(r, 19))
	if err != nil || string(got) != "abcdefabcdefabcdefa" {
		t.Fatalf("got %q: %v", got, err)
	}
}

func TestRepeatedContentChunkBoundaries(t *testing.T) {
	r := io.LimitReader(&repeatedContent{pattern: "abc"}, 1001)
	var result strings.Builder
	buf := make([]byte, 7)
	for {
		n, err := r.Read(buf)
		result.Write(buf[:n])
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if got, want := result.String(), strings.Repeat("abc", 334)[:1001]; got != want {
		t.Fatal("repetition changed at chunk boundaries")
	}
}

func TestGenerateRejectsInvalidSizes(t *testing.T) {
	for _, r := range []*GenerateRequest{{Size: -1}, {SizeInMb: -1}, {Lines: -1}, {Size: 1, FileCount: -1}, {Size: 1, Prefix: "long"}} {
		r.Dest = &location.Resource{URL: "mem://fixture"}
		if err := r.Validate(); err == nil {
			t.Fatalf("accepted %#v", r)
		}
	}
}
