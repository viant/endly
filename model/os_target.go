package model

// OsTarget represents operating system target
type OsTarget struct {
	System             string
	Architecture       string
	Name               string
	MinRequiredVersion string
	MaxAllowedVersion  string
}
