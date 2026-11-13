package model

import (
	"github.com/viant/toolbox/data"
)

// ModifiedStateEvent represent modified state event
type ModifiedStateEvent struct {
	Variables Variables
	In        map[string]interface{}
	Modified  map[string]interface{}
}

// NewModifiedStateEvent creates a new modified state event.
func NewModifiedStateEvent(variables Variables, in, out data.Map) *ModifiedStateEvent {
	var result = &ModifiedStateEvent{
		Variables: variables,
		In:        make(map[string]interface{}),
		Modified:  make(map[string]interface{}),
	}

	for _, variable := range variables {
		name := data.ExtractPath(variable.Name)
		if variable.KeyFrom != "" {
			result.Modified[name], _ = out.LookupValue(name)
			continue
		}
		from := data.ExtractPath(variable.From)
		result.In[from], _ = in.GetValue(from)
		result.Modified[name], _ = out.GetValue(name)
	}
	return result
}
