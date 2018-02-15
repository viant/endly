package endly

import "fmt"

//BuildGoal builds goal represents a build goal
type BuildGoal struct {
	Name                string              `required:"true"`
	InitTransfers       *StorageCopyRequest ` description:"files transfer before build command"`
	Command             *ExtractableCommand `required:"true"  description:"build command"`
	PostTransfers       *StorageCopyRequest ` description:"files transfer after build command"`
	VerificationCommand *ExtractableCommand
}

//BuildMeta build meta provides instruction how to build an app
type BuildMeta struct {
	Name         string                  `required:"true" description:"name of build system"`
	Goals        []*BuildGoal            `required:"true" description:"build goals"`
	Dependencies []*DeploymentDependency `description:"deployment dependencies"`
	goalsIndex   map[string]*BuildGoal
}

//Validate validates build meta.
func (m *BuildMeta) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("metaBuild.Names %v", m.Name)

	}
	if len(m.Goals) == 0 {
		return fmt.Errorf("metaBuild.Goals were empty %v", m.Name)
	}
	return nil
}
