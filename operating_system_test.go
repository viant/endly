package endly_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly"
	"testing"
)

func TestOperatingSystem_Matches(t *testing.T) {

	operatingSystem := &endly.OperatingSystem{
		System:       "darwin",
		Name:         "maxosx",
		Hardware:     "x86_64",
		Architecture: "x86_64",
		Version:      "16.7.0",
	}
	assert.True(t, operatingSystem.Matches(&endly.OperatingSystemTarget{System: "darwin"}))
	assert.False(t, operatingSystem.Matches(&endly.OperatingSystemTarget{System: "linux"}))

	assert.True(t, operatingSystem.Matches(&endly.OperatingSystemTarget{Name: "maxosx"}))
	assert.False(t, operatingSystem.Matches(&endly.OperatingSystemTarget{Name: "ubuntu"}))

	assert.True(t, operatingSystem.Matches(&endly.OperatingSystemTarget{MinRequiredVersion: "16.7.0"}))
	assert.False(t, operatingSystem.Matches(&endly.OperatingSystemTarget{MinRequiredVersion: "17.0.0"}))

	assert.True(t, operatingSystem.Matches(&endly.OperatingSystemTarget{MaxAllowedVersion: "16.7.0"}))

	assert.True(t, operatingSystem.Matches(&endly.OperatingSystemTarget{MaxAllowedVersion: "15.7.0"}))
	assert.False(t, operatingSystem.Matches(&endly.OperatingSystemTarget{MaxAllowedVersion: "18.0.0"}))

}
