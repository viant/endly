package endly

import (
	"github.com/pkg/errors"
	"github.com/viant/toolbox/url"
)

//DeploymentMetaRequest represents DeploymentMeta register request.
type DeploymentMetaRequest struct {
	Source *url.Resource
}

//DeploymentMetaResponse represents deployment response
type DeploymentMetaResponse struct {
	Meta *DeploymentMeta
}

//DeploymentMeta represents description of deployment instructions for various operating system
type DeploymentMeta struct {
	Name       string //app name
	Versioning string //versioning system, i.e. Major.Minor.Release
	Targets    []*DeploymentTargetMeta
}

//DeploymentTargetMeta represents specific instruction for given os deployment.
type DeploymentTargetMeta struct {
	Version           string                  //version of the software
	MinReleaseVersion map[string]string       //min release version, key is major.minor, value is release or update version
	OsTarget          *OperatingSystemTarget  //if specified matches current os
	Deployment        *Deployment             //actual deployment instruction
	Dependencies      []*DeploymentDependency //app dependencies like sdk
}

//Deployment represents deployment instruction
type Deployment struct {
	Pre          *DeploymentAddition
	Transfer     *Transfer           //actual copy instruction
	Command      *ExtractableCommand //post deployment command like tar xvzf
	VersionCheck *ExtractableCommand //command to check version
	Post         *DeploymentAddition //post deployment
}

//DeploymentAddition represents deployment additions.
type DeploymentAddition struct {
	SuperUser bool
	Commands  []string
	Transfers []*Transfer
}

//Validate checks if request if valid
func (d *Deployment) Validate() error {

	if d.Transfer == nil {
		return errors.New("Transfer was nil")
	}
	if d.Transfer.Target == nil {
		return errors.New("Transfer.Target was not specified")
	}
	if d.Transfer.Target.URL == "" {
		return errors.New("Transfer.Target.URL was empty")
	}
	if d.Transfer.Source.URL == "" {
		return errors.New("Transfer.Source.URL was empty")
	}
	return nil

}

//Validate checks is meta is valid.
func (m *DeploymentMeta) Validate() error {
	if len(m.Targets) == 0 {
		return errors.New("Targets were empty")
	}
	for _, target := range m.Targets {
		if target.Deployment == nil {
			return errors.New("Target.Deployment was empty")
		}
		err := target.Deployment.Validate()
		if err != nil {
			return err
		}
	}
	return nil
}

//AsCommandRequest creates a command request.
func (a *DeploymentAddition) AsCommandRequest() *CommandRequest {
	return &CommandRequest{
		Commands:  a.Commands,
		SuperUser: a.SuperUser,
	}
}

//MatchVersion checks expected and actual version returns true if matches.
func MatchVersion(expected, actual string) bool {
	var expectedLength = len(expected)
	var actualLength = len(actual)
	if expectedLength == 0 || actualLength == 0 {
		return true
	}

	if actualLength == expectedLength {
		return expected == actual
	}
	if actualLength > expectedLength {
		actual = string(actual[:expectedLength])
	} else {
		expected = string(expected[:actualLength])
	}
	return expected == actual
}

//Match provides build instruction for matching os and version
func (m *DeploymentMeta) Match(operatingSystem *OperatingSystem, requestedVersion string) *DeploymentTargetMeta {
	for _, candidate := range m.Targets {
		if candidate.Version != "" {
			if !MatchVersion(requestedVersion, candidate.Version) {
				continue
			}
		}
		if operatingSystem.Matches(candidate.OsTarget) {
			return candidate
		}
	}
	return nil
}
