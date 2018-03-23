package workflow

import (
	"github.com/viant/endly"
	"github.com/viant/neatly"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
	"github.com/viant/endly/model"
)

var endlyRemoteRepo = "https://raw.githubusercontent.com/viant/endly/master/%v"
var endlyLocalRepo = "mem://github.com/viant/endly/%v"

//Dao represents a workflow loader
type Dao struct {
	Dao *neatly.Dao
}

//Load loads workflow into memory
func (d *Dao) Load(context *endly.Context, source *url.Resource) (*model.Workflow, error) {
	resource, err := context.ExpandResource(source)
	if err != nil {
		return nil, err
	}
	result := &model.Workflow{}
	var state = data.NewMap()
	err = d.Dao.Load(state, resource, result)
	if err == nil {
		d.Dao.AddStandardUdf(context.State())
		err = result.Validate()
	}

	return result, err
}

//NewRepoResource returns new woorkflow repo resource, it takes context map and resource URI
func (d *Dao) NewRepoResource(context data.Map, URI string) (*url.Resource, error) {
	var resource, err = d.Dao.NewRepoResource(context, URI)
	return resource, err
}

//NewDao returns a new NewDao
func NewDao() *Dao {
	return &Dao{
		Dao: neatly.NewDao(true, endlyLocalRepo, endlyRemoteRepo, "", nil),
	}
}
