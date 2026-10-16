package webdriver

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/tebeka/selenium"
	"testing"
)

type tableDataElement struct{ selenium.WebElement }

func (*tableDataElement) TagName() (string, error) { return "table", nil }
func (*tableDataElement) GetAttribute(name string) (string, error) {
	return "", nil // outerHTML is a DOM property, not an HTML attribute.
}
func (*tableDataElement) GetProperty(name string) (string, error) {
	if name != "outerHTML" {
		return "", fmt.Errorf("unexpected property %s", name)
	}
	return `<table><thead><tr><th>Count</th></tr></thead><tbody><tr><td>12</td></tr></tbody></table>`, nil
}
func TestTableDataAliasDispatch(t *testing.T) {
	for _, method := range []string{"Data", "TableData"} {
		response := &ServiceCallResponse{}
		require.NoError(t, (&service{}).callMethod(&tableDataElement{}, method, response, []interface{}{"objects"}))
		require.Equal(t, []map[string]interface{}{{"Count": "12"}}, response.Result[0])
	}
}
