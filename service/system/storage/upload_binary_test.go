package storage

import (
	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	"github.com/viant/endly/model/location"
	"os"
	"path/filepath"
	"testing"
)

func TestUploadBinaryPreservesBytesAndDoesNotExpandVariables(t *testing.T) {
	ctx := endly.New().NewContext(nil)
	defer ctx.Close()
	state := ctx.State()
	payload := []byte{0x89, 'P', 'N', 'G', 0, '$', 's', 'e', 'c', 'r', 'e', 't', 0xff}
	state.Put("asset", payload)
	state.Put("secret", "must-not-be-inserted")
	path := filepath.Join(t.TempDir(), "screenshot.png")
	response := &UploadResponse{}
	require.NoError(t, endly.Run(ctx, &UploadRequest{SourceKey: "asset", Dest: location.NewResource(path)}, response))
	actual, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, payload, actual)
	require.Equal(t, len(payload), response.Size)
}
