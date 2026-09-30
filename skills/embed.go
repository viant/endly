// Package skills contains the versioned Endly skill catalog.
package skills

import "embed"

// Files is the catalog shipped with the executable.
//go:embed */SKILL.md */references/*
var Files embed.FS
