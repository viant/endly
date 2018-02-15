package endly

import (
	"github.com/pkg/errors"
	"github.com/viant/toolbox/url"
)

//DeploymentLoadMetaRequest represents DeploymentMeta register request.
type DeploymentLoadMetaRequest struct {
	Source *url.Resource `required:"true" description:"deployment meta location"`
}

//DeploymentLoadMetaResponse represents deployment response
type DeploymentLoadMetaResponse struct {
	Meta *DeploymentMeta
}

//DeploymentMeta represents description of deployment instructions for various operating system
type DeploymentMeta struct {
	Name       string                                                                                                                     //app name
	Versioning string                  `required:"true" description:"versioning template for dynamic discovery i.e. Major.Minor.Release"` //versioning system, i.e. Major.Minor.Release
	Targets    []*DeploymentTargetMeta `required:"true" description:"deployment instruction for various version and operating systems"`
}

//DeploymentTargetMeta represents specific instruction for given os deployment.
type DeploymentTargetMeta struct {
	Version           string                                                                                                                              //version of the software
	MinReleaseVersion map[string]string       `required:"true" description:"min release version, key is major.minor, value is release or update version"` //min release version, key is major.minor, value is release or update version
	OsTarget          *OperatingSystemTarget  `description:"operating system match"`                                                                      //if specified matches current os
	Deployment        *Deployment             `required:"true" description:"actual deployment instructions"`                                              //actual deployment instruction
	Dependencies      []*DeploymentDependency `description:"app dependencies like sdk"`                                                                   //app dependencies like sdk
}

//Deployment represents deployment instruction
type Deployment struct {
	Pre          *DeploymentAddition `description:"initialization deployment instruction"`
	Transfer     *Transfer           `required:"true" description:"software deployment instruction"` //actual copy instruction
	Command      *ExtractableCommand `description:"post deployment commands, i.e. tar xvzf"`         //post deployment command like tar xvzf
	VersionCheck *ExtractableCommand `description:"version extraction command"`                      //command to check version
	Post         *DeploymentAddition `description:"post deployment instruction"`
}

//DeploymentAddition represents deployment additions.
type DeploymentAddition struct {
	SuperUser bool
	Commands  []string    `description:"os command"`
	Transfers []*Transfer `description:"asset transfer"`
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
	if d.Transfer.Source == nil {
		return errors.New("Transfer.Source was empty")
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
