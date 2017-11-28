package sso


type StaticRoute struct {
	URI string
	Directory string
}

type Config struct {
	Port string
	StaticRoutes []*StaticRoute
}




