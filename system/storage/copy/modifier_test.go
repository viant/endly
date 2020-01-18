package copy

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/afs/file"
	"github.com/viant/afs/matcher"
	"github.com/viant/endly"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNewModifier(t *testing.T) {

	ctx := endly.New().NewContext(nil)
	now := time.Now()

	var useCases = []struct {
		description string
		when        *Matcher
		replacement map[string]string
		expand      bool
		info        os.FileInfo
		modTime     time.Time
		state       map[string]interface{}
		text        string
		expect      string
		expectError bool
	}{

		{
			description: "replace modifier",
			replacement: map[string]string{
				"foo": "bar",
			},
			text:   " foo is foodd",
			expect: " bar is bardd",
		},
		{
			description: "state modifier",
			replacement: map[string]string{
				"foo": "bar",
			},
			state: map[string]interface{}{
				"msg": "secret",
			},
			expand: true,
			text:   " foo is ${msg}a",
			expect: " bar is secreta",
		},
		{
			description: "state modifier - no expand",
			replacement: map[string]string{
				"foo": "bar",
			},
			state: map[string]interface{}{
				"msg": "secret",
			},
			expand: false,
			text:   " foo is ${msg}a",
			expect: " bar is ${msg}a",
		},

		{
			description: "replace only, do not expand due to binary data ",
			replacement: map[string]string{
				"foo": "bar",
			},
			state: map[string]interface{}{
				"msg": "secret",
			},
			expand: true,
			text:   " foo is ${msg}\b5Ὂg̀9! ℃ᾭG",
			expect: " bar is ${msg}\b5Ὂg̀9! ℃ᾭG",
		},

		{
			description: "no change  - no replacement matched",
			replacement: map[string]string{
				"foo": "bar",
			},
			text:   "test",
			expect: "test",
		},
		{
			description: "no change - file to large",
			replacement: map[string]string{
				"foo": "bar",
			},
			text:   strings.Repeat("foo ", 1024*1024),
			expect: strings.Repeat("foo ", 1024*1024),
		},
		{
			description: "no change  - file no matched",
			replacement: map[string]string{
				"foo": "bar",
			},
			when: &Matcher{
				Basic: &matcher.Basic{Suffix: ".json"},
			},
			info:   file.NewInfo("test.txt", 4, 0644, now, false),
			text:   "foo is great",
			expect: "foo is great",
		},
		{
			description: "changed  - file matched",
			replacement: map[string]string{
				"foo": "bar",
			},
			when: &Matcher{
				Basic: &matcher.Basic{Suffix: ".txt"},
			},
			info:   file.NewInfo("test.txt", 4, 0644, now, false),
			text:   "foo is great",
			expect: "bar is great",
		},
		{
			description: "no change - emoty content",
			replacement: map[string]string{
				"foo": "bar",
			},
			expand: true,
			text:   "",
			expect: "",
		},
		{
			description: "no change  - file no matched - outside modification window",
			replacement: map[string]string{
				"foo": "bar",
			},
			when: &Matcher{
				UpdatedBefore: "hourAgo",
			},
			info:   file.NewInfo("test.txt", 4, 0644, now, false),
			text:   "foo is great",
			expect: "foo is great",
		},
		{
			description: "changed  - file matched - with modification window",
			replacement: map[string]string{
				"foo": "bar",
			},
			when: &Matcher{
				Basic:        &matcher.Basic{Suffix: ".txt"},
				UpdatedAfter: "hourAgo",
			},
			info:   file.NewInfo("test.txt", 4, 0644, now, false),
			text:   "foo is great",
			expect: "bar is great",
		},
		{
			description: "error invalid after expression",
			replacement: map[string]string{
				"foo": "bar",
			},
			when: &Matcher{
				Basic:        &matcher.Basic{Suffix: ".txt"},
				UpdatedAfter: "bladh",
			},
			info:        file.NewInfo("test.txt", 4, 0644, now, false),
			text:        "foo is great",
			expectError: true,
		},
		{
			description: "error invalid before expression",
			replacement: map[string]string{
				"foo": "bar",
			},
			when: &Matcher{
				Basic:         &matcher.Basic{Suffix: ".txt"},
				UpdatedBefore: "bladh",
			},
			info:        file.NewInfo("test.txt", 4, 0644, now, false),
			text:        "foo is great",
			expectError: true,
		},
	}

	for _, useCase := range useCases {
		if useCase.modTime.IsZero() {
			useCase.modTime = now
		}
		if useCase.info == nil {
			useCase.info = file.NewInfo("test.txt", int64(len(useCase.text)), 0644, useCase.modTime, false)
		}
		if len(useCase.state) > 0 {
			state := ctx.State()
			for k, v := range useCase.state {
				state.Put(k, v)
			}
		}

		matcher, err := NewModifier(ctx, useCase.when, useCase.replacement, useCase.expand)
		if useCase.expectError {
			assert.NotNil(t, err, useCase.description)
			continue
		}
		if !assert.Nil(t, err, useCase.description) {
			continue
		}


		readerCloser := ioutil.NopCloser(strings.NewReader(useCase.text))
		_, reader, err := matcher(useCase.info, readerCloser)
		assert.Nil(t, err, useCase.description)
		actual, err := ioutil.ReadAll(reader)
		assert.Nil(t, err, useCase.description)
		assert.EqualValues(t, useCase.expect, string(actual), useCase.description)
	}

}
