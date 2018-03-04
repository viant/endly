package deploy

import (
	"errors"
	"github.com/viant/endly"
	"github.com/viant/endly/system/exec"
	"github.com/viant/endly/system/storage"
	"github.com/viant/toolbox/url"
)

//Request represent a deploy request
type Request struct {
	Target  *url.Resource `required:"true" description:"target host"`                                                                                   //target host
	MetaURL string        `description:"optional URL for meta deployment file, if left empty the meta URL is construct as meta/deployment/**AppName**"` //deployment URL for meta deployment instruction
	AppName string        `required:"true" description:"application name, as defined in meta deployment file"`                                          //app name
	Version string        `description:"min required version, it can be 1, or 1.2 or specific version 1.2.1"`                                           //requested version
	Force   bool          `description:"force deployment even if app has been already installed"`                                                       //flag force deployment, by default if requested version matches the one from command version check. deployment is skipped.
}

//Validate check if request is valid otherwise returns error.
func (r *Request) Validate() error {
	if r.Target == nil {
		return errors.New("target host was nil")
	}
	if r.AppName == "" {
		return errors.New("app name was empty")
	}
	return nil
}

//Response represents a deploy response.
type Response struct {
	Version string
}

//LoadMetaRequest represents Meta register request.
type LoadMetaRequest struct {
	Source *url.Resource `required:"true" description:"deployment meta location"`
}

//LoadMetaResponse represents deployment response
type LoadMetaResponse struct {
	Meta *Meta
}

//Meta represents description of deployment instructions for various operating system
type Meta struct {
	Name       string        //app name
	Versioning string        `required:"true" description:"versioning template for dynamic discovery i.e. Major.Minor.Release"` //versioning system, i.e. Major.Minor.Release
	Targets    []*TargetMeta `required:"true" description:"deployment instruction for various version and operating systems"`
}

//Dependency represents deployment dependency
type Dependency struct {
	Name    string
	Version string
}

//TargetMeta represents specific instruction for given os deployment.
type TargetMeta struct {
	Version           string                       //version of the software
	MinReleaseVersion map[string]string            `required:"true" description:"min release version, key is major.minor, value is release or update version"` //min release version, key is major.minor, value is release or update version
	OsTarget          *endly.OperatingSystemTarget `description:"operating system match"`                                                                      //if specified matches current os
	Deployment        *Deployment                  `required:"true" description:"actual deployment instructions"`                                              //actual deployment instruction
	Dependencies      []*Dependency                `description:"app dependencies like sdk"`                                                                   //app dependencies like sdk
}

//Deployment represents deployment instruction
type Deployment struct {
	Pre          *Addition                `description:"initialization deployment instruction"`
	Transfer     *storage.Transfer        `required:"true" description:"software deployment instruction"` //actual copy instruction
	Command      *exec.ExtractableCommand `description:"post deployment commands, i.e. tar xvzf"`         //post deployment command like tar xvzf
	VersionCheck *exec.ExtractableCommand `description:"version extraction command"`                      //command to check version
	Post         *Addition                `description:"post deployment instruction"`
}

//Addition represents deployment additions.
type Addition struct {
	SuperUser bool
	Commands  []string            `description:"os command"`
	Transfers []*storage.Transfer `description:"asset transfer"`
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
func (m *Meta) Validate() error {
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

//AsExtractRequest creates a command request.
func (a *Addition) AsCommandRequest() *exec.RunRequest {
	return &exec.RunRequest{
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
func (m *Meta) Match(operatingSystem *endly.OperatingSystem, requestedVersion string) *TargetMeta {
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
