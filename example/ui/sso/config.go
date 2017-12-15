package sso

import "github.com/viant/dsc"

type StaticRoute struct {
	URI string
	Directory string
}

type Config struct {
	Port string
	StaticRoutes []*StaticRoute
	DsConfig *dsc.Config
}




