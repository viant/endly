package git

import (
	"fmt"
	"github.com/viant/endly/model/location"
)

// CheckoutRequest represents checkout request. If target directory exist and contains matching origin URL,
// only taking the latest changes without overriding local if performed, otherwise full checkout
type CheckoutRequest struct {
	Origin *location.Resource `required:"true" description:"checkout source for git or svn or simply file::/path"`
	Dest   *location.Resource `required:"true" description:"checkout dest defined by host and path URL"`
	Depth  int
}

// CheckoutResponse represents checkout response
type CheckoutResponse StatusResponse

// StatusResponse represents version control status response
type StatusResponse struct {
	*Info
}

// Info represents version control info
type Info struct {
	IsVersionControlManaged bool   //returns true if directory is source controlled managed
	Origin                  string //Origin URL
	Revision                string //Origin Revision
	Branch                  string //current branch
	IsUptoDate              bool
	Added                   []string //new files
	Untracked               []string //untracked files
	Modified                []string //modified files
	Deleted                 []string //deleted files
}

// StatusRequest represents version control status
type StatusRequest struct {
	Source *location.Resource `required:"true"`
}

// CommitRequest represents a commit request
type CommitRequest struct {
	Source      *location.Resource `required:"true" description:"location to local source code"`
	Message     string             `required:"true"`
	Credentials string
}

// CommitResponse represents a commit response
type CommitResponse struct {
	*Info
}

// Init initializes request
func (r *CheckoutRequest) Init() error {
	if r.Origin == nil {
		return nil
	}
	if r.Dest == nil {
		return nil
	}
	return nil
}

// Validate validates request
func (r *CheckoutRequest) Validate() error {
	if r.Origin == nil {
		return fmt.Errorf("origin was empty")
	}
	if r.Dest == nil {
		return fmt.Errorf("dest was empty")
	}
	if r.Dest.Scheme() != "file" {
		return fmt.Errorf("unsupported dest scheme: %v, supported scheme: file", r.Dest.Scheme())
	}
	return nil
}

// Init initializes request
func (r *CommitRequest) Init() error {
	return nil
}

// Validate validates request
func (r *CommitRequest) Validate() error {
	if r.Source == nil {
		return fmt.Errorf("source was empty")
	}
	if r.Message == "" {
		return fmt.Errorf("message was empty")
	}
	return nil
}

// HasPendingChanges returns true if there are any untracked, new, modified, deleted files.
func (r *Info) HasPendingChanges() bool {
	return len(r.Added) > 0 || len(r.Untracked) > 0 || len(r.Deleted) > 0 || len(r.Modified) > 0
}

// Init initializes request
func (r *StatusRequest) Init() error {
	return nil
}

// Validate validates request
func (r *StatusRequest) Validate() error {
	if r.Source == nil {
		return fmt.Errorf("source type was empty")
	}
	return nil
}

// NewInfo create new info
func NewInfo() *Info {
	return &Info{
		Added:     make([]string, 0),
		Untracked: make([]string, 0),
		Modified:  make([]string, 0),
		Deleted:   make([]string, 0),
	}
}
