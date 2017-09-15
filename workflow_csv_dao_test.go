package endly_test

import (
	"testing"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"path"
	"fmt"
	"github.com/stretchr/testify/assert"
)

func TestNewWorkflowDao(t *testing.T) {

	fileName, _, _ := toolbox.CallerInfo(2)
	parent, _ := path.Split(fileName)


	dao := endly.NewWorkflowDao()
	conext := &endly.Context{Context: toolbox.NewContext()}
	workflow, err := dao.Load(conext, &endly.Resource{
		URL:fmt.Sprintf("file://%v", path.Join(parent, "test/workflow/w1.csv")),
	})
	assert.Nil(t, err)
	assert.NotNil(t, workflow)
	assert.Equal(t, "Simple Test", workflow.Name)
	assert.Equal(t, "My description", workflow.Description)

	assert.Equal(t, 3, len(workflow.Tasks))
	assert.Equal(t, "Simple Http Test", workflow.Tasks[0].Name)
	assert.Equal(t, 1, len(workflow.Tasks[0].Variables))
	assert.Equal(t, "v10", workflow.Tasks[0].Variables[0].Name)
	assert.Equal(t, 2, len(workflow.Tasks[1].Variables))

	assert.Equal(t, "v30", workflow.Tasks[2].Variables[0].Name)



}
