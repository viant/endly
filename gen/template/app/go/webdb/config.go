package webdb

import (
	"github.com/viant/dsc"
	"github.com/viant/toolbox/url"
)

//Config representa an application config
type Config struct {
	Port      int
	Datastore *dsc.Config
}

//NewConfig creates a new app config
func NewConfig(port int, config *dsc.Config) *Config {
	return &Config{Port: port, Datastore: config}
}

//NewConfig creates a new app config from supplied URL
func NewConfigFromURL(URL string) (*Config, error) {
	var result = &Config{}
	var resource = url.NewResource(URL)
	return result, resource.Decode(result)
}
