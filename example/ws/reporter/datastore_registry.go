package reporter

import (
	"fmt"
	"github.com/viant/dsc"
)

//DatastoreRegistry represents a datastore datastores
type DatastoreRegistry map[string]dsc.Manager

//Register register datastore config with datastore connectivity config.
func (r *DatastoreRegistry) Register(config *DatastoreConfig) error {
	manager, err := dsc.NewManagerFactory().Create(config.Config)
	if err != nil {
		return fmt.Errorf("failed to create datastore manager for %v, %v", config.Name, err)
	}
	(*r)[config.Name] = manager
	return nil
}
