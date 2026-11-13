package exec

import (
	"fmt"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	"github.com/viant/endly/model"
	"github.com/viant/toolbox/data"
)

func warmSystemMetadata(t *testing.T) (*endly.Context, *model.Session) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX local shell")
	}
	parent := endly.New().NewContext(nil)
	t.Cleanup(parent.Close)
	request := NewRunRequest(nil, false, "true")
	request.StrictCompletion = true
	request.CheckError = true
	require.NoError(t, New().Run(parent, request).Err)
	session, err := TerminalSession(parent, GetServiceTarget(nil))
	require.NoError(t, err)
	state := parent.State()
	state.SetValue("os.custom", data.Map{"enabled": false, "limit": 0, "optional": nil})
	return parent, session
}

func systemMetadata(context *endly.Context) data.Map {
	state := context.State()
	return state.GetMap("os")
}

func TestSystemMetadataDetachesWarmContextClone(t *testing.T) {
	parent, session := warmSystemMetadata(t)
	original := systemMetadata(parent)
	username := original["user"]
	child := parent.Clone()
	childSession := *session
	childSession.Username = "isolated-child"
	service := &execService{}
	service.updateSystemInfo(child.State(), &childSession)
	require.Equal(t, username, original["user"], "exec metadata must not mutate the warmed parent's os map")
	require.Equal(t, username, systemMetadata(parent)["user"])
	require.Equal(t, "isolated-child", systemMetadata(child)["user"])
	require.Equal(t, original["custom"], systemMetadata(child)["custom"])
	require.Equal(t, session.Service.HardwareInfo().Arch, systemMetadata(child)["arch"])
	require.Equal(t, session.Service.OsInfo().System, systemMetadata(child)["system"])
}

func TestSystemMetadataConcurrentWarmContextClones(t *testing.T) {
	parent, session := warmSystemMetadata(t)
	original := systemMetadata(parent)
	username := original["user"]
	const workers = 16
	contexts := make([]*endly.Context, workers)
	for i := range contexts {
		contexts[i] = parent.Clone()
	}
	start := make(chan struct{})
	var group sync.WaitGroup
	for i, child := range contexts {
		group.Add(1)
		go func(i int, child *endly.Context) {
			defer group.Done()
			childSession := *session
			childSession.Username = fmt.Sprintf("worker-%d", i)
			<-start
			for n := 0; n < 200; n++ {
				(&execService{}).updateSystemInfo(child.State(), &childSession)
			}
		}(i, child)
	}
	close(start)
	group.Wait()
	require.Equal(t, username, original["user"])
	for i, child := range contexts {
		require.Equal(t, fmt.Sprintf("worker-%d", i), systemMetadata(child)["user"])
		require.Equal(t, original["custom"], systemMetadata(child)["custom"])
	}
}
