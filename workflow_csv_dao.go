package endly

import (
	"fmt"
	"github.com/viant/neatly"
	"os"
	"github.com/viant/toolbox/url"
	"github.com/viant/toolbox/data"
)

var endlyRemoteRepo = "https://raw.githubusercontent.com/viant/endly/master/%v"
var endlyLocalRepo = fmt.Sprintf("file://%v/src/github.com/viant/endly/%v", os.Getenv("GOPATH"), "%v")

type WorkflowDao struct {
	Dao *neatly.Dao
}

func (d *WorkflowDao) Load(context *Context, source *url.Resource) (*Workflow, error) {
	fmt.Printf("Loading : %v\n", source.URL)

	resource, err := context.ExpandResource(source)
	if err != nil {
		return nil, err
	}
	result := &Workflow{}
	var state = context.state
	state.DisableUDF()
	defer state.EnableUDF()
	err = d.Dao.Load(context.state, resource, result)
	return result, err
}

func (d *WorkflowDao) NewRepoResource(context data.Map, URI string) (*url.Resource, error) {

	fmt.Printf("URI: %v\n", URI)
	var resource, err = d.Dao.NewRepoResource(context, URI)

	fmt.Printf("EXPANDED  %v %v\n", resource.URL, err)
	return resource, err
}

func NewWorkflowDao() *WorkflowDao {
	return &WorkflowDao{
		Dao: neatly.NewDao(endlyLocalRepo, endlyRemoteRepo, "", nil),
	}
}
