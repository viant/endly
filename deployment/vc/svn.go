package vc

import (
	"fmt"
	"github.com/viant/toolbox/url"
	"path"
	"strings"
	"github.com/viant/endly"
	"github.com/viant/endly/system/exec"
	"github.com/viant/endly/util"
)

type svnService struct{}

func (s *svnService) checkInfo(context *endly.Context, request *StatusRequest) (*StatusResponse, error) {
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
				Command: fmt.Sprintf("svn info"),
				Extraction: []*endly.DataExtraction{
					{
						RegExpr: "^URL:[\\t\\s]+([^\\s]+)",
						Key:     "origin",
					},
					{
						RegExpr: "Revision:\\s+([^\\s]+)",
						Key:     "revision",
					},
				},
			},
			{
				Command: fmt.Sprintf("svn stat"),
			},
		},
	})

	if err != nil {
		return nil, err
	}
	if revison, has := response.Extracted["revision"]; has {
		result.Revision = revison
	}
	if origin, has := response.Extracted["origin"]; has {
		result.Origin = origin
		_, result.Branch = path.Split(origin)
	}
	if strings.Contains(response.Stdout(1), "is not a working copy") {
		return result, nil
	}
	result.IsVersionControlManaged = true

	readSvnStatus(response, result.Info)
	return result, nil
}

func readSvnStatus(commandResult *exec.CommandResponse, response *Info) {
	response.New = make([]string, 0)
	response.Modified = make([]string, 0)
	response.Deleted = make([]string, 0)
	response.Untracked = make([]string, 0)
	for _, line := range strings.Split(commandResult.Stdout(), "\n") {
		if len(line) == 0 {
			continue
		}
		columns, ok := util.ExtractColumns(line)
		if !ok || len(columns) < 2 {
			continue
		}
		file := columns[1]
		switch columns[0] {
		case "?":
			response.Untracked = append(response.Untracked, file)
		case "A":
			response.New = append(response.New, file)
		case "D":
			response.Deleted = append(response.Deleted, file)
		case "M":
			response.Modified = append(response.Modified, file)
		}
	}
	if len(response.Modified)+len(response.Deleted)+len(response.New) == 0 {
		response.IsUptoDate = true
	}
}

func (s *svnService) pull(context *endly.Context, request *PullRequest) (*PullResponse, error) {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	var response = &PullResponse{
		&Info{},
	}
	return response, s.runSecureSvnCommand(context, target, request.Origin, response.Info, "up")
}

func (s *svnService) checkout(context *endly.Context, request *CheckoutRequest) (*Info, error) {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	var vcInfo = &Info{}
	err = s.runSecureSvnCommand(context, target, request.Origin, vcInfo, "co", request.Origin.URL, target.DirectoryPath())
	return vcInfo, err
}

func (s *svnService) runSecureSvnCommand(context *endly.Context, target *url.Resource, origin *url.Resource, info *Info, command string, arguments ...string) error {
	username, _, err := origin.LoadCredential(true)
	if err != nil {
		return err
	}
	var credentials = make(map[string]string)
	credentials[CredentialKey] = origin.Credential
	_, err = exec.Execute(context, target, &exec.ExtractableCommand{
		Options: &exec.ExecutionOptions{
			TimeoutMs:   1000 * 200,
			Terminators: []string{"Password for", "(yes/no)?"},
		},
		Executions: []*exec.Execution{
			{
				Command: fmt.Sprintf("svn %v --username=%v %v", command, username, strings.Join(arguments, " ")),
				Errors:  []string{"No such file or directory", "Event not found", "Errors validating server certificate", "Unable to connect to a repository"},
			},
			{
				Credentials: credentials,
				MatchOutput: "Password",
				Command:     CredentialKey,
				Errors:      []string{"No such file or directory", "Event not found", "Username:"},
			},
			{
				MatchOutput: "Store password unencrypted",
				Command:     "no",
				Errors:      []string{"No such file or directory", "Event not found", "Errors validating server certificate"},
			},
		},
	})
	err = checkVersionControlAuthErrors(err, origin)
	if err != nil {
		return err
	}
	response, err := s.checkInfo(context, &StatusRequest{
		Target: target,
		Type:   "svn",
	})
	if response != nil {
		*info = *response.Info
	}
	return err
}

func (s *svnService) commit(context *endly.Context, request *CommitRequest) (*CommitResponse, error) {
	runResponse, err := exec.Execute(context, request.Target, &exec.ExtractableCommand{
		Executions: []*exec.Execution{
			{
				Command: fmt.Sprintf("svn ci -m \"%v\" ", strings.Replace(request.Message, "\"", "'", len(request.Message))),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if util.CheckNoSuchFileOrDirectory(runResponse.Stdout()) {
		return nil, fmt.Errorf("failed to commit %v", runResponse.Stdout())
	}

	var response = &CommitResponse{}
	statusResponse, err := s.checkInfo(context, &StatusRequest{
		Target: request.Target,
	})
	if err != nil {
		return nil, err
	}
	response.Info = statusResponse.Info
	return response, nil
}
