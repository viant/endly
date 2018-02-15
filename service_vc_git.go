package endly

import (
	"fmt"
	"github.com/lunixbochs/vtclean"
	"github.com/viant/toolbox/url"
	"path"
	"strings"
)

type gitService struct{}

const (
	newFile          = "new file:"
	deletedFile      = "deleted:"
	modifiedFile     = "modified:"
	expectChangeType = iota
	expectedUnTrackedFile
)

func addIfMatched(line, fragment string, result *[]string) {
	matchedPosition := strings.Index(line, fragment)
	if matchedPosition != -1 {
		newFile := strings.TrimSpace(string(line[matchedPosition+len(fragment):]))
		*result = append(*result, newFile)
	}

}

func extractGitStatus(stdout string, response *VcInfo) {
	if strings.Contains(stdout, "nothing to commit") {
		response.IsUptoDate = true
	}

	response.New = make([]string, 0)
	response.Modified = make([]string, 0)
	response.Deleted = make([]string, 0)
	response.Untracked = make([]string, 0)
	var state = expectChangeType

	for _, line := range strings.Split(stdout, "\r\n") {
		line = vtclean.Clean(line, false)
		switch state {
		case expectChangeType:
			addIfMatched(line, newFile, &response.New)
			addIfMatched(line, modifiedFile, &response.Modified)
			addIfMatched(line, deletedFile, &response.Deleted)
			if strings.Contains(line, "Untracked files:") {
				state = expectedUnTrackedFile
			}

		case expectedUnTrackedFile:
			hintsPosition := strings.Index(line, "(")
			if hintsPosition != -1 {
				continue
			}
			candidate := strings.Trim(line, " \t")
			if len(candidate) > 0 {
				response.Untracked = append(response.Untracked, candidate)
			}
		}
	}
}

func extractRevision(stdout string, response *VcInfo) {
	if strings.Contains(stdout, "unknown revision") {
		response.IsVersionControlManaged = true
		return
	}
	response.Revision = strings.TrimSpace(stdout)

}

func (s *gitService) checkInfo(context *Context, request *VcStatusRequest) (*VcStatusResponse, error) {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	var result = &VcStatusResponse{&VcInfo{}}
	response, err := context.Execute(target, fmt.Sprintf("cd %v", target.DirectoryPath()))
	if err != nil || CheckCommandNotFound(response.Stdout()) {
		return result, nil
	}

	response, err = context.Execute(request.Target, &ExtractableCommand{
		Executions: []*Execution{

			{
				Command: fmt.Sprintf("git status"),
				Extraction: []*DataExtraction{{
					RegExpr: "On branch[\\s\\t]+([^\\s]+)",
					Key:     "branch",
				}},
			},
			{
				Command: fmt.Sprintf("git remote -v"),
				Extraction: []*DataExtraction{{
					RegExpr: "origin[\\s\\t]+([^\\s]+)\\s+\\(fetch\\)",
					Key:     "origin",
				}},
			},
			{
				Command: fmt.Sprintf("git rev-parse HEAD"),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if branch, has := response.Extracted["branch"]; has {
		result.Branch = branch
	}
	if origin, has := response.Extracted["origin"]; has {
		result.Origin = origin
	}

	if strings.Contains(response.Stdout(), "Not a git") {
		return result, nil
	}
	result.IsVersionControlManaged = true
	extractGitStatus(response.Stdout(0), result.VcInfo)
	extractRevision(response.Stdout(2), result.VcInfo)
	return result, nil
}

func (s *gitService) pull(context *Context, request *VcPullRequest) (*VcPullResponse, error) {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	origin, err := context.ExpandResource(request.Origin)
	if err != nil {
		return nil, err
	}

	var response = &VcPullResponse{
		VcInfo: &VcInfo{},
	}
	_, err = context.Execute(target, fmt.Sprintf("cd %v", target.DirectoryPath()))
	if err != nil {
		return nil, err
	}
	return response, s.runSecureCommand(context, request.Type, origin, target, "git pull", response.VcInfo, false)
}

func (s *gitService) checkout(context *Context, request *VcCheckoutRequest) (*VcInfo, error) {
	origin, err := context.ExpandResource(request.Origin)
	if err != nil {
		return nil, err
	}
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	var username, _, _ = origin.LoadCredential(false)
	if origin.Credential != "" && username == "" {
		return nil, fmt.Errorf("username was empty %v, %v", origin.URL, origin.Credential)
	}
	var parent, projectName = path.Split(target.DirectoryPath())
	var useParentDirectory = true
	var _, originProjectName = path.Split(origin.DirectoryPath())
	if originProjectName == projectName {
		projectName = "."
		_, err = context.Execute(target, []string{fmt.Sprintf("mkdir -p %v", target.DirectoryPath())})
		if err != nil {
			return nil, err
		}
		useParentDirectory = false
	} else {
		_, err = context.Execute(target, fmt.Sprintf("cd %v", parent))
		if err != nil {
			return nil, err
		}
	}

	var info = &VcInfo{}
	err = s.runSecureCommand(context, request.Type, origin, target, fmt.Sprintf("git clone %v %v", origin.CredentialURL(username, ""), projectName), info, useParentDirectory)
	return info, err
}

func (s *gitService) runSecureCommand(context *Context, versionControlType string, origin, target *url.Resource, command string, info *VcInfo, useParentDirectory bool) (err error) {
	var credentials = make(map[string]string)
	credentials[versionControlCredentialKey] = origin.Credential
	commandTarget, _ := context.ExpandResource(target)
	if useParentDirectory {
		commandTarget.Rename("")
	}
	_, err = context.Execute(commandTarget, &ExtractableCommand{
		Options: &ExecutionOptions{
			TimeoutMs:   1000 * 200,
			Terminators: []string{"Password"},
		},
		Executions: []*Execution{
			{
				Command: command,
				Errors:  []string{"No such file or directory", "Event not found", "Unable to connect"},
			},
			{
				Credentials: credentials,
				MatchOutput: "Password",
				Command:     versionControlCredentialKey,
				Errors:      []string{"No such file or directory", "Event not found", "Authentication failed"},
			},
		},
	})

	err = checkVersionControlAuthErrors(err, origin)
	if err != nil {
		return err
	}
	response, err := s.checkInfo(context, &VcStatusRequest{
		Target: target,
		Type:   versionControlType,
	})

	if err != nil {
		return err
	}
	*info = *response.VcInfo
	return nil
}

func (s *gitService) commit(context *Context, request *VcCommitRequest) (*VcCommitResponse, error) {
	checkInfo, err := s.checkInfo(context, &VcStatusRequest{
		Target: request.Target,
	})
	if err != nil {
		return nil, err
	}
	if len(checkInfo.Untracked) > 0 {
		for _, file := range checkInfo.Untracked {
			_, err = context.Execute(request.Target, &ExtractableCommand{
				Executions: []*Execution{
					{
						Command: fmt.Sprintf("git add %v ", file),
						Errors:  []string{"No such file or directory", "Errors"},
					},
				},
			})
			if err != nil {
				return nil, err
			}
		}
	}
	message := strings.Replace(request.Message, "\"", "'", len(request.Message))
	_, err = context.Execute(request.Target, &ExtractableCommand{
		Executions: []*Execution{
			{
				Command: fmt.Sprintf("git commit -m \"%v\" -a", message),
				Errors:  []string{"No such file or directory", "Errors"},
			},
		},
	})
	_, err = context.Execute(request.Target, &ExtractableCommand{
		Executions: []*Execution{
			{
				Command: "git push",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	respons, err := s.checkInfo(context, &VcStatusRequest{
		Target: request.Target,
	})
	if err != nil {
		return nil, err
	}
	return &VcCommitResponse{respons.VcInfo}, nil
}
