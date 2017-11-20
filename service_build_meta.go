package endly

import "fmt"





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
	Name             string
	Goals            []*BuildGoal
	Dependencies 	 []*DeploymentDependency
	goalsIndex       map[string]*BuildGoal
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
