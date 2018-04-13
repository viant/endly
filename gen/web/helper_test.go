package web

import (
	"testing"
	"github.com/viant/toolbox/storage"
	"strings"
	"github.com/stretchr/testify/assert"
)

func TestDownload(t *testing.T) {
	var srv = storage.NewMemoryService()
	srv.Upload("mem://dd01/folder/file1.txt", strings.NewReader("abc1"))
	srv.Upload("mem://dd01/folder/file2.txt", strings.NewReader("abc2"))
	srv.Upload("mem://dd01/folder/sub/file.txt", strings.NewReader("abc3"))

	assets, err := DownloadAll("mem://dd01/folder")
	if assert.Nil(t, err) {
		if assert.True(t, len(assets) > 0) {

			assert.EqualValues(t, "abc1", assets["file1.txt"])
			assert.EqualValues(t, "abc2", assets["file2.txt"])
			assert.EqualValues(t, "abc3", assets["sub/file.txt"])
		}
	}

}
