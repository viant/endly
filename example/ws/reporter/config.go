package reporter

import "github.com/viant/dsc"

type DatastoreConfig struct {
	Name   string
	Config *dsc.Config
}

type Config struct {
	RepositoryDatastore string
	Datastores          []*DatastoreConfig
}
