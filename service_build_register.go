package endly

import "fmt"

//BuildRegisterMetaRequest represents a BuildMeta register request.
type BuildRegisterMetaRequest struct {
	Meta *BuildMeta
}

//BuildRegisterMetaResponse represents a BuildMeta register response.
type BuildRegisterMetaResponse struct {
	Name string
}

//OperatingSystemDeployment represents specific instruction for given os deplyoment.
type OperatingSystemDeployment struct {
	OsTarget *OperatingSystemTarget
	Deploy   *DeploymentDeployRequest
}

//BuildGoal builds goal represents a build goal
type BuildGoal struct {
	Name                string
	InitTransfers       *TransferCopyRequest
	Command             *ManagedCommand
	PostTransfers       *TransferCopyRequest
	VerificationCommand *ManagedCommand
}

//BuildMeta build meta provides instruction how to build an app
type BuildMeta struct {
	Sdk              string
	SdkVersion       string
	Name             string
	Goals            []*BuildGoal
	goalsIndex       map[string]*BuildGoal
	BuildDeployments []*OperatingSystemDeployment //defines deployment of the build app itself, i.e how to get maven installed
}

//Validate validates build meta.
func (m *BuildMeta) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("MetaBuild.Names %v", m.Name)

	}
	if len(m.Goals) == 0 {
		return fmt.Errorf("MetaBuild.Goals were empty %v", m.Name)
	}
	return nil
}

//Match provides build instruction for matching os and version
func (m *BuildMeta) Match(operatingSystem *OperatingSystem, version string) *OperatingSystemDeployment {
	for _, candidate := range m.BuildDeployments {
		osTarget := candidate.OsTarget
		if version != "" {
			if candidate.Deploy.Transfer.Target.Version != version {
				continue
			}
		}
		if operatingSystem.Matches(osTarget) {
			return candidate
		}
	}
	return nil
}
