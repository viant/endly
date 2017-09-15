package endly_test

import (
	"testing"
	"github.com/viant/endly"
	"github.com/stretchr/testify/assert"
)


func TestWorkflowService_Run(t *testing.T) {

	manager := endly.NewManager()
	srv, err := manager.Service(endly.WorkflowServiceId)
	assert.Nil(t, err)
	assert.NotNil(t, srv)



}


