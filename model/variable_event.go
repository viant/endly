package model

import (
	"github.com/viant/toolbox/data"
)

//ModifiedStateEvent represent modified state event
type ModifiedStateEvent struct {
	Variables Variables
	In        map[string]interface{}
	Modified  map[string]interface{}
}

//NewModifiedStateEvent creates a new modified state event.
func NewModifiedStateEvent(variables Variables, in, out data.Map) *ModifiedStateEvent {
	var result = &ModifiedStateEvent{
		Variables: variables,
		In:        make(map[string]interface{}),
		Modified:  make(map[string]interface{}),
	}

	for _, variable := range variables {
		from := data.ExtractPath(variable.From)
		result.In[from], _ = in.GetValue(from)
		name := data.ExtractPath(variable.Name)
		result.Modified[name], _ = out.GetValue(name)
	}
	return result
}
