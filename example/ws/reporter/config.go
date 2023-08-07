package reporter

import "github.com/viant/dsc"

// DatastoreConfig represents datastore config
type DatastoreConfig struct {
	Name   string
	Config *dsc.Config
}

// Config represents areporter config
type Config struct {
	RepositoryDatastore string
	Datastores          []*DatastoreConfig
}
