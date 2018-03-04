package vc

import (
	"fmt"
	"github.com/lunixbochs/vtclean"
	"github.com/viant/endly"
	"github.com/viant/endly/system/exec"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox/url"
	"path"
	"strings"
)

type git struct{}

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

func extractGitStatus(stdout string, response *Info) {
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

func extractRevision(stdout string, response *Info) {
	if strings.Contains(stdout, "unknown revision") {
		response.IsVersionControlManaged = true
		return
	}
	response.Revision = strings.TrimSpace(stdout)

}

func (s *git) checkInfo(context *endly.Context, request *StatusRequest) (*StatusResponse, error) {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	var result = &StatusResponse{&Info{}}
	response, err := exec.Execute(context, target, fmt.Sprintf("cd %v", target.DirectoryPath()))
	if err != nil || util.CheckCommandNotFound(response.Stdout()) {
		return result, nil
	}

	response, err = exec.Execute(context, request.Target, &exec.ExtractableCommand{
		Executions: []*exec.Execution{

			{
				Command: fmt.Sprintf("git status"),
				Extraction: []*endly.DataExtraction{{
					RegExpr: "On branch[\\s\\t]+([^\\s]+)",
					Key:     "branch",
				}},
			},
			{
				Command: fmt.Sprintf("git remote -v"),
				Extraction: []*endly.DataExtraction{{
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
	extractGitStatus(response.Stdout(0), result.Info)
	extractRevision(response.Stdout(2), result.Info)
	return result, nil
}

func (s *git) pull(context *endly.Context, request *PullRequest) (*PullResponse, error) {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	origin, err := context.ExpandResource(request.Origin)
	if err != nil {
		return nil, err
	}

	var response = &PullResponse{
		Info: &Info{},
	}
	_, err = exec.Execute(context, target, fmt.Sprintf("cd %v", target.DirectoryPath()))
	if err != nil {
		return nil, err
	}
	return response, s.runSecureCommand(context, request.Type, origin, target, "git pull", response.Info, false)
}

func (s *git) checkout(context *endly.Context, request *CheckoutRequest) (*Info, error) {
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
		_, err = exec.Execute(context, target, []string{fmt.Sprintf("mkdir -p %v", target.DirectoryPath())})
		if err != nil {
			return nil, err
		}
		useParentDirectory = false
	} else {
		_, err = exec.Execute(context, target, fmt.Sprintf("cd %v", parent))
		if err != nil {
			return nil, err
		}
	}

	var info = &Info{}
	err = s.runSecureCommand(context, request.Type, origin, target, fmt.Sprintf("git clone %v %v", origin.CredentialURL(username, ""), projectName), info, useParentDirectory)
	return info, err
}

func (s *git) runSecureCommand(context *endly.Context, versionControlType string, origin, target *url.Resource, command string, info *Info, useParentDirectory bool) (err error) {
	var credentials = make(map[string]string)
	credentials[CredentialKey] = origin.Credential
	commandTarget, _ := context.ExpandResource(target)
	if useParentDirectory {
		commandTarget.Rename("")
	}
	_, err = exec.Execute(context, commandTarget, &exec.ExtractableCommand{
		Options: &exec.ExecutionOptions{
			TimeoutMs:   1000 * 200,
			Terminators: []string{"Password"},
		},
		Executions: []*exec.Execution{
			{
				Command: command,
				Errors:  []string{"No such file or directory", "Event not found", "Unable to connect"},
			},
			{
				Credentials: credentials,
				MatchOutput: "Password",
				Command:     CredentialKey,
				Errors:      []string{"No such file or directory", "Event not found", "Authentication failed"},
			},
		},
	})

	err = checkVersionControlAuthErrors(err, origin)
	if err != nil {
		return err
	}
	response, err := s.checkInfo(context, &StatusRequest{
		Target: target,
		Type:   versionControlType,
	})

	if err != nil {
		return err
	}
	*info = *response.Info
	return nil
}

func (s *git) commit(context *endly.Context, request *CommitRequest) (*CommitResponse, error) {
	checkInfo, err := s.checkInfo(context, &StatusRequest{
		Target: request.Target,
	})
	if err != nil {
		return nil, err
	}
	if len(checkInfo.Untracked) > 0 {
		for _, file := range checkInfo.Untracked {
			_, err = exec.Execute(context, request.Target, &exec.ExtractableCommand{
				Executions: []*exec.Execution{
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
	_, err = exec.Execute(context, request.Target, &exec.ExtractableCommand{
		Executions: []*exec.Execution{
			{
				Command: fmt.Sprintf("git commit -m \"%v\" -a", message),
				Errors:  []string{"No such file or directory", "Errors"},
			},
		},
	})
	_, err = exec.Execute(context, request.Target, &exec.ExtractableCommand{
		Executions: []*exec.Execution{
			{
				Command: "git push",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	respons, err := s.checkInfo(context, &StatusRequest{
		Target: request.Target,
	})
	if err != nil {
		return nil, err
	}
	return &CommitResponse{respons.Info}, nil
}
