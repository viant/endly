package dsunit

import (
	"github.com/stretchr/testify/require"
	"github.com/viant/toolbox/data"
	"testing"
)

func TestAutoGenerateShortExpressionsReturnErrors(t *testing.T) {
	for _, expression := range []string{"", "$", "x", "${"} {
		t.Run(expression, func(t *testing.T) {
			fixture := &TableData{AutoGenerate: map[string]string{"id": expression}}
			require.NotPanics(t, func() { require.Error(t, fixture.AutoGenerateIfNeeded(data.NewMap())) })
		})
	}
}
