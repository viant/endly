package endly_test

import (
	"github.com/viant/endly"
	"os"
	"testing"
)

func Test_Renderer_PrintTable(t *testing.T) {

	render := endly.NewRenderer(os.Stdout, 80)

	var data = [][]string{
		{
			"Id", "true", "Lorem Ipsum is simply dummy text ",
		},
		{
			"Name", "false", "Lorem Ipsum is simply dummy text of the printing and typesetting ",
		},
	}
	render.PrintTable("table1", []string{"name", "required", "description"}, data, 80)

}
