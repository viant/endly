package workflow

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/endly"
	"github.com/viant/endly/model"
)

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
