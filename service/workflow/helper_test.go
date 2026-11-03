package workflow

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	"github.com/viant/endly/model"
)

func TestWorkflowStackReadsDoNotInitializeContext(t *testing.T) {
	for name, read := range map[string]func(*endly.Context) *model.Process{"last": Last, "lastWorkflow": LastWorkflow} {
		t.Run(name, func(t *testing.T) {
			ctx := endly.New().NewContext(nil)
			defer ctx.Close()
			require.Nil(t, read(ctx))
			require.False(t, ctx.Contains(processesKey), "reading an absent stack must not publish a new context entry")
			require.NotContains(t, ctx.AsyncUnsafeKeys, processesKey, "read-only inspection must not race with context cloning")
		})
	}
}

func TestAsyncContextKeepsAnIndependentWorkflowStack(t *testing.T) {
	parent := endly.New().NewContext(nil)
	root := model.NewProcess(nil, &model.Workflow{}, nil)
	Push(parent, root)
	left, right := parent.Clone(), parent.Clone()
	left.MakeAsyncSafe()
	right.MakeAsyncSafe()
	l := model.NewProcess(nil, &model.Workflow{}, root)
	r := model.NewProcess(nil, &model.Workflow{}, root)
	Push(left, l)
	Push(right, r)
	require.Same(t, root, Last(parent), "a worker must not change its parent's active workflow")
	require.Same(t, l, Last(left), "goto must resolve against its own worker")
	require.Same(t, r, Last(right))
	require.Same(t, l, Pop(left))
	require.Nil(t, Last(left))
	require.Same(t, r, Last(right), "worker completion must not pop a sibling workflow")
	require.Same(t, root, Last(parent))
}
