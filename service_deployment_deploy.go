package endly

import (
	"fmt"
	"github.com/viant/toolbox"
	"strings"
)

//DeploymentDeployRequest represent a deploy request
type DeploymentDeployRequest struct {
	Sdk          string
	SdkVersion   string
	Pre          *DeploymentAddition
	Transfer     *Transfer           //actual instalation transfer
	Command      *ManagedCommand     //post deployment check like tar xvzf
	VersionCheck *ManagedCommand     //command to run version
	Post         *DeploymentAddition //post deployment
	AppName      string              //app name
	Force        bool                //flag force deployment, by default if requested version (Transfer.Target.Version is the one from command version check. deployment is skipped.
}

//DeploymentDeployResponse represents a deploy response.
type DeploymentDeployResponse struct {
	Version string
}

//DeploymentAddition represents deployment additions.
type DeploymentAddition struct {
	SuperUser bool
	Commands  []string
	Transfers []*Transfer
}

//Validate checks if request if valid
func (r *DeploymentDeployRequest) Validate() error {

	if r.Transfer == nil {
		return fmt.Errorf("Failed to deploy app, transfer was nil")
	}
	if r.Transfer.Target == nil {
		return fmt.Errorf("Failed to deploy app, target was not specified")
	}
	if r.Transfer.Target.URL == "" {
		return fmt.Errorf("Failed to deploy app, target URL was empty")
	}
	if r.Transfer.Source.URL == "" {
		return fmt.Errorf("Failed to deploy app, Source URL was empty")
	}
	if r.AppName == "" {
		_, appName := toolbox.URLSplit(r.Transfer.Source.URL)
		var versionPosition = strings.LastIndex(appName, "-")
		if versionPosition != -1 {
			appName = string(appName[:versionPosition])
		}
		r.AppName = appName
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
