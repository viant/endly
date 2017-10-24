package endly



//VcInfo represents version control info
type VcInfo struct {
	IsVersionControlManaged bool//returns true if directory is source controlled managed
	Origin                  string//Origin URL
	Revision                string//Origin Revision
	Branch                  string//current branch
	IsUptoDate              bool
	New                     []string//new files
	Untracked               []string//untracked files
	Modified                []string//modified files
	Deleted                 []string//deleted files
}


//HasPendingChanges returns true if there are any untracked, new, modified, deleted files.
func (r *VcInfo) HasPendingChanges() bool {
	return len(r.New) > 0 || len(r.Untracked) > 0 || len(r.Deleted) > 0 || len(r.Modified) > 0
}
