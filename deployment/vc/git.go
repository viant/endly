package vc

import (
	"fmt"
	"github.com/lunixbochs/vtclean"
	"github.com/viant/endly"
	"github.com/viant/endly/system/exec"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox/secret"
	"github.com/viant/toolbox/url"
	"path"
	"strings"
	"github.com/viant/endly/model"
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
	target, err := context.ExpandResource(request.Source)
	if err != nil {
		return nil, err
	}
	var result = &StatusResponse{&Info{}}
	var runResponse = &exec.RunResponse{}
	if err := endly.Run(context, exec.NewRunRequest(target, false, fmt.Sprintf("cd %v", target.DirectoryPath())), runResponse); err != nil || util.CheckCommandNotFound(runResponse.Stdout()) {
		return result, nil
	}

	runRequest := exec.NewExtractRequest(request.Source, exec.DefaultOptions(),
		exec.NewExtractCommand(fmt.Sprintf("git status"), "", nil, nil,
			model.NewExtract("branch", "On branch[\\s\\t]+([^\\s]+)", true)),

		exec.NewExtractCommand(fmt.Sprintf("git remote -v"), "", nil, nil,
			model.NewExtract("origin", "origin[\\s\\t]+([^\\s]+)\\s+\\(fetch\\)", true)),
		exec.NewExtractCommand(fmt.Sprintf("git rev-parse HEAD"), "", nil, nil))

	if err = endly.Run(context, runRequest, runResponse); err != nil {
		return nil, err
	}

	if branch, has := runResponse.Data["branch"]; has {
		result.Branch = branch.(string)
	}
	if origin, has := runResponse.Data["origin"]; has {
		result.Origin = origin.(string)
	}
	if util.EscapedContains(strings.ToLower(runResponse.Stdout()), "not a git") {
		result.IsVersionControlManaged = false
		return result, nil
	}

	result.IsVersionControlManaged = true
	extractGitStatus(runResponse.Stdout(0), result.Info)
	extractRevision(runResponse.Stdout(2), result.Info)
	return result, nil
}

func (s *git) pull(context *endly.Context, request *PullRequest) (*PullResponse, error) {
	target, err := context.ExpandResource(request.Dest)
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

	if err = endly.Run(context, exec.NewRunRequest(target, false, fmt.Sprintf("cd %v", target.DirectoryPath())), nil); err != nil {
		return nil, err
	}
	return response, s.runSecureCommand(context, request.Type, origin, target, "git pull", response.Info, false)
}

func (s *git) checkout(context *endly.Context, request *CheckoutRequest) (*Info, error) {
	origin, err := context.ExpandResource(request.Origin)
	if err != nil {
		return nil, err
	}
	dest, err := context.ExpandResource(request.Dest)
	if err != nil {
		return nil, err
	}

	username := ""
	if origin.Credentials != "" {
		username, err = util.GetUsername(context.Secrets, origin.Credentials)
		if err != nil {
			return nil, err
		}
	}

	var parent, projectName = path.Split(dest.DirectoryPath())
	var useParentDirectory = true
	var _, originProjectName = path.Split(origin.DirectoryPath())
	if originProjectName == projectName {
		projectName = "."
		if dest.DirectoryPath() != "/" {
			if err := endly.Run(context, exec.NewRunRequest(dest, false, fmt.Sprintf("mkdir -p %v", dest.DirectoryPath())), nil); err != nil {
				return nil, nil
			}
		}
		useParentDirectory = false
	} else {
		if err := endly.Run(context, exec.NewRunRequest(dest, false, fmt.Sprintf("cd %v", parent)), nil); err != nil {
			return nil, err
		}
	}

	var info = &Info{}
	err = s.runSecureCommand(context, request.Type, origin, dest, fmt.Sprintf("git clone %v %v", origin.CredentialURL(username, ""), projectName), info, useParentDirectory)
	return info, err
}

func (s *git) runSecureCommand(context *endly.Context, versionControlType string, origin, source *url.Resource, command string, info *Info, useParentDirectory bool) (err error) {
	var secrets = make(map[string]string)
	secrets[CredentialKey] = origin.Credentials
	commandTarget, _ := context.ExpandResource(source)
	if useParentDirectory {
		commandTarget.Rename("")
	}

	var runRequest = exec.NewExtractRequest(commandTarget,
		exec.DefaultOptions(),
		exec.NewExtractCommand(command, "", nil, []string{util.NoSuchFileOrDirectory, "Event not found", "Unable to connect"}),
		exec.NewExtractCommand(CredentialKey, "Password", nil, []string{util.NoSuchFileOrDirectory, "Event not found", "Authentication failed"}),
	)
	runRequest.Secrets = secret.NewSecrets(secrets)
	runRequest.TimeoutMs = 1000 * 200
	runRequest.Terminators = append(runRequest.Terminators, "Password")

	if err = endly.Run(context, runRequest, nil); err != nil {
		err = checkVersionControlAuthErrors(err, context.Secrets, origin)
	}

	if err != nil {
		return err
	}

	response, err := s.checkInfo(context, &StatusRequest{
		Source: source,
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
		Source: request.Source,
	})
	if err != nil {
		return nil, err
	}
	if len(checkInfo.Untracked) > 0 {
		for _, file := range checkInfo.Untracked {
			var runRequest = exec.NewExtractRequest(request.Source,
				exec.DefaultOptions(),
				exec.NewExtractCommand(fmt.Sprintf("git add %v ", file), "", nil, []string{util.NoSuchFileOrDirectory, "Errors"}),
			)
			if err = endly.Run(context, runRequest, nil); err != nil {
				return nil, err
			}
		}
	}
	message := strings.Replace(request.Message, "\"", "'", len(request.Message))

	var runRequest = exec.NewExtractRequest(request.Source,
		exec.DefaultOptions(),
		exec.NewExtractCommand(fmt.Sprintf("git commit -m \"%v\" -a", message), "", nil, []string{util.NoSuchFileOrDirectory, "Errors"}))

	if err = endly.Run(context, runRequest, nil); err == nil {
		err = endly.Run(context, exec.NewRunRequest(request.Source, false, "git push"), nil)
	}
	if err != nil {
		return nil, err
	}
	response, err := s.checkInfo(context, &StatusRequest{
		Source: request.Source,
	})
	if err != nil {
		return nil, err
	}
	return &CommitResponse{response.Info}, nil
}
