package sso

import "github.com/viant/dsc"

//StaticRoute represent a static route
type StaticRoute struct {
	URI       string
	Directory string
}

//Config represents sso config
type Config struct {
	Port         string
	StaticRoutes []*StaticRoute
	DsConfig     *dsc.Config
}
