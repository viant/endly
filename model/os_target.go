package model


//OsTarget represents operating system target
type OsTarget struct {
	System             string
	Name               string
	MinRequiredVersion string
	MaxAllowedVersion  string
}

