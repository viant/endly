package endly

import (
	"github.com/viant/toolbox"
	"strings"
)

//OperatingSystem represents an OperatingSystem
type OperatingSystem struct {
	Family  string
	Name    string
	Version string
	Path    *SystemPath
}

func normalizeVersion(version string) int {
	var result = 0
	if strings.Contains(version, ".") {
		var fragments = strings.Split(version, ".")
		for i, fragment := range fragments {
			factor := 10 ^ (len(fragments) - i + 1)
			result += toolbox.AsInt(fragment) * factor
		}
	}
	return result
}

//Matches returns true if operating system matches provided target
func (s *OperatingSystem) Matches(target *OperatingSystemTarget) bool {
	if target == nil {
		return true
	}
	if target.Name != "" && target.Name != s.Name {
		return false
	}
	if target.Family != "" && target.Family != s.Family {
		return false
	}

	if target.MinRequiredVersion == "" && target.MaxAllowedVersion == "" {
		return true
	}
	var actualVersion = normalizeVersion(s.Version)

	if target.MinRequiredVersion != "" {
		var minRequiredVersion = normalizeVersion(target.MinRequiredVersion)
		if actualVersion < minRequiredVersion {
			return false
		}
	}
	var maxAllowedVersion = normalizeVersion(target.MaxAllowedVersion)
	return actualVersion > maxAllowedVersion
}

//SystemPath represents a system path
type SystemPath struct {
	index      map[string]bool
	SystemPath []string
	Path       []string
}

//Push appends path to the system paths
func (p *SystemPath) Push(paths ...string) {
	for _, path := range paths {
		if strings.Contains(path, "\n") {
			continue
		}
		if _, has := p.index[path]; has {
			return
		}
		p.Path = append(p.Path, path)
		p.index[path] = true
	}
}

//EnvValue returns evn values
func (p *SystemPath) EnvValue() string {
	var directories = append(p.Path, p.SystemPath...)
	return strings.Join(directories, ":")
}

//OperatingSystemTarget represents operating system target
type OperatingSystemTarget struct {
	Family             string
	Name               string
	MinRequiredVersion string
	MaxAllowedVersion  string
}
