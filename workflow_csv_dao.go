package endly

import (
	"fmt"
	"github.com/viant/neatly"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
	"os"
)

var endlyRemoteRepo = "https://raw.githubusercontent.com/viant/endly/master/%v"
var endlyLocalRepo = fmt.Sprintf("file://%v/src/github.com/viant/endly/%v", os.Getenv("GOPATH"), "%v")

//WorkflowDao represents a workflow loader
type WorkflowDao struct {
	Dao *neatly.Dao
}

//Load loads workflow into memory
func (d *WorkflowDao) Load(context *Context, source *url.Resource) (*Workflow, error) {
	resource, err := context.ExpandResource(source)
	if err != nil {
		return nil, err
	}
	result := &Workflow{}
	var state = data.NewMap()
	err = d.Dao.Load(state, resource, result)
	if err == nil {
		d.Dao.AddStandardUdf(context.state)
	}
	return result, err
}

//NewRepoResource returns new woorkflow repo resource, it takes context map and resource URI
func (d *WorkflowDao) NewRepoResource(context data.Map, URI string) (*url.Resource, error) {
	var resource, err = d.Dao.NewRepoResource(context, URI)
	return resource, err
}

//NewWorkflowDao returns a new NewWorkflowDao
func NewWorkflowDao() *WorkflowDao {
	return &WorkflowDao{
		Dao: neatly.NewDao(endlyLocalRepo, endlyRemoteRepo, "", nil),
	}
}
