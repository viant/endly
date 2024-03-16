package model

import (
	"github.com/viant/gosh"
	"github.com/viant/toolbox"
	"math"
	"strings"
)

// Os represents an Os
type OperatingSystem struct {
	*gosh.OSInfo
	*gosh.HardwareInfo
}

// Matches returns true if operating system matches provided target
func (s *OperatingSystem) Matches(target *OsTarget) bool {
	if target == nil {
		return true
	}
	if target.Name != "" && target.Name != s.Name {
		return false
	}
	if target.System != "" && target.System != s.System {
		return false
	}

	if target.MinRequiredVersion == "" && target.MaxAllowedVersion == "" {
		return true
	}
	var versionFragmentCount = strings.Count(s.Version, ".")
	var actualVersion = normalizeVersion(s.Version, versionFragmentCount)

	if target.MinRequiredVersion != "" {
		var minRequiredVersion = normalizeVersion(target.MinRequiredVersion, versionFragmentCount)
		if actualVersion < minRequiredVersion {
			return false
		}
	}
	var maxAllowedVersion = normalizeVersion(target.MaxAllowedVersion, versionFragmentCount)
	return actualVersion >= maxAllowedVersion
}

func normalizeVersion(version string, count int) int {
	var result = 0
	var fragments = strings.Split(version, ".")
	for i, fragment := range fragments {
		factor := math.Pow(10.0, (2.0 * float64(count-i)))
		result += toolbox.AsInt(fragment) * int(factor)
	}
	return result
}
