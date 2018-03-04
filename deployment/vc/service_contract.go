package vc

import (
	"fmt"
	"github.com/viant/toolbox/url"
)

//CheckoutRequest represents checkout request. If target directory exist and contains matching origin URL,
// only taking the latest changes without overriding local if performed, otherwise full checkout
type CheckoutRequest struct {
	Type               string        `required:"true" description:"version control type: git, svn"`
	Target             *url.Resource `required:"true" description:"checkout source  defined by host and path URL"`
	Origin             *url.Resource `required:"true"`
	Modules            []string      `description:"list of modules to checkout"`
	RemoveLocalChanges bool          `description:"flat to remove local directory before checkout"`
}

//CheckoutResponse represents checkout response
type CheckoutResponse struct {
	Checkouts map[string]*Info
}

//Init initializes request
func (r *CheckoutRequest) Init() error {
	versionControlRequestInit(r.Origin, &r.Type)
	return nil
}

//Validate validates request
func (r *CheckoutRequest) Validate() error {

	if r.Origin == nil {
		return fmt.Errorf("origin type was empty")
	}
	if r.Target == nil {
		return fmt.Errorf("target type was empty")
	}
	if r.Type == "" {
		return fmt.Errorf("version control type was empty for %v", r.Origin.URL)
	}
	return nil
}



//CommitRequest represents a commit request
type CommitRequest struct {
	Target  *url.Resource `required:"true" description:"location to local source code"`
	Type    string        `description:"version control type: git,svn"`
	Message string        `required:"true"`
}

//CommitResponse represents a commit response
type CommitResponse struct {
	*Info
}

//Init initializes request
func (r *CommitRequest) Init() error {
	return versionControlRequestInit(r.Target, &r.Type)
}

//Validate validates request
func (r *CommitRequest) Validate() error {
	if r.Target == nil {
		return fmt.Errorf("target type was empty")
	}
	if r.Type == "" {
		return fmt.Errorf("type was empty for %v", r.Target.URL)
	}
	return nil
}


//Info represents version control info
type Info struct {
	IsVersionControlManaged bool   //returns true if directory is source controlled managed
	Origin                  string //Origin URL
	Revision                string //Origin Revision
	Branch                  string //current branch
	IsUptoDate              bool
	New                     []string //new files
	Untracked               []string //untracked files
	Modified                []string //modified files
	Deleted                 []string //deleted files
}

//HasPendingChanges returns true if there are any untracked, new, modified, deleted files.
func (r *Info) HasPendingChanges() bool {
	return len(r.New) > 0 || len(r.Untracked) > 0 || len(r.Deleted) > 0 || len(r.Modified) > 0
}


//PullRequest represents a pull request
type PullRequest struct {
	Type   string
	Target *url.Resource `required:"true"`
	Origin *url.Resource `required:"true"` //version control origin
}

//Init initializes request
func (r *PullRequest) Init() error {
	return versionControlRequestInit(r.Target, &r.Type)
}

//Validate validates request
func (r *PullRequest) Validate() error {
	if r.Target == nil {
		return fmt.Errorf("target type was empty")
	}
	if r.Type == "" {
		return fmt.Errorf("type was empty for %v", r.Target.URL)
	}
	return nil
}

//PullResponse represents a pull response
type PullResponse struct {
	*Info
}


//StatusRequest represents version control status
type StatusRequest struct {
	Target *url.Resource `required:"true"`
	Type   string
}

//Init initializes request
func (r *StatusRequest) Init() error {
	return versionControlRequestInit(r.Target, &r.Type)
}

//Validate validates request
func (r *StatusRequest) Validate() error {
	if r.Target == nil {
		return fmt.Errorf("target type was empty")
	}
	if r.Type == "" {
		return fmt.Errorf("type was empty for %v", r.Target.URL)
	}
	return nil
}

//StatusResponse represents version control status response
type StatusResponse struct {
	*Info
}

