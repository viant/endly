package model_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/endly/model"
	"testing"

)

func TestOperatingSystem_Matches(t *testing.T) {

	operatingSystem := &model.OperatingSystem{
		System:       "darwin",
		Name:         "maxosx",
		Hardware:     "x86_64",
		Architecture: "x86_64",
		Version:      "16.7.0",
	}
	assert.True(t, operatingSystem.Matches(&model.OsTarget{System: "darwin"}))
	assert.False(t, operatingSystem.Matches(&model.OsTarget{System: "linux"}))

	assert.True(t, operatingSystem.Matches(&model.OsTarget{Name: "maxosx"}))
	assert.False(t, operatingSystem.Matches(&model.OsTarget{Name: "ubuntu"}))

	assert.True(t, operatingSystem.Matches(&model.OsTarget{MinRequiredVersion: "16.7.0"}))
	assert.False(t, operatingSystem.Matches(&model.OsTarget{MinRequiredVersion: "17.0.0"}))

	assert.True(t, operatingSystem.Matches(&model.OsTarget{MaxAllowedVersion: "16.7.0"}))

	assert.True(t, operatingSystem.Matches(&model.OsTarget{MaxAllowedVersion: "15.7.0"}))
	assert.False(t, operatingSystem.Matches(&model.OsTarget{MaxAllowedVersion: "18.0.0"}))

}
