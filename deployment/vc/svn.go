package vc

import (
	"fmt"
	"github.com/viant/endly"
	"github.com/viant/endly/model"
	"github.com/viant/endly/system/exec"
	"github.com/viant/endly/util"
	"github.com/viant/toolbox/secret"
	"github.com/viant/toolbox/url"
	"path"
	"strings"
)

type svnService struct{}

func (s *svnService) checkInfo(context *endly.Context, request *StatusRequest) (*StatusResponse, error) {
	target, err := context.ExpandResource(request.Source)
	if err != nil {
		return nil, err
	}
	var result = &StatusResponse{&Info{}}
	var runResponse = &exec.RunResponse{}
	if err = endly.Run(context, exec.NewRunRequest(target, false, fmt.Sprintf("cd %v", target.DirectoryPath())), runResponse); err != nil ||
		util.CheckCommandNotFound(runResponse.Stdout()) {
		return result, nil
	}

	extractRequest := exec.NewExtractRequest(target,
		exec.DefaultOptions(),
		exec.NewExtractCommand(fmt.Sprintf("svn info"), "", nil, nil,
			model.NewExtract("origin", "^URL:[\\t\\s]+([^\\s]+)", false),
			model.NewExtract("revision", "Revision:\\s+([^\\s]+)", false)),
		exec.NewExtractCommand(fmt.Sprintf("svn stat"), "", nil, nil))

	if err = endly.Run(context, extractRequest, runResponse); err != nil {
		return nil, err
	}

	if revison, has := runResponse.Data["revision"]; has {
		result.Revision = revison.(string)
	}
	if origin, has := runResponse.Data["origin"]; has {
		result.Origin = origin.(string)
		_, result.Branch = path.Split(result.Origin)
	}
	if strings.Contains(runResponse.Stdout(1), "is not a working copy") {
		return result, nil
	}
	result.IsVersionControlManaged = true

	readSvnStatus(runResponse, result.Info)
	return result, nil
}

func readSvnStatus(commandResult *exec.RunResponse, response *Info) {
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
	target, err := context.ExpandResource(request.Dest)
	if err != nil {
		return nil, err
	}
	var response = &PullResponse{
		&Info{},
	}
	return response, s.runSecureSvnCommand(context, target, request.Origin, response.Info, "up")
}

func (s *svnService) checkout(context *endly.Context, request *CheckoutRequest) (*Info, error) {
	dest, err := context.ExpandResource(request.Dest)
	if err != nil {
		return nil, err
	}
	var vcInfo = &Info{}
	err = s.runSecureSvnCommand(context, dest, request.Origin, vcInfo, "co", request.Origin.URL, dest.DirectoryPath())
	return vcInfo, err
}

func (s *svnService) runSecureSvnCommand(context *endly.Context, source *url.Resource, origin *url.Resource, info *Info, command string, arguments ...string) error {
	var username, err = util.GetUsername(context.Secrets, origin.Credentials)
	if err != nil {
		return err
	}
	var secrets = make(map[string]string)
	secrets[CredentialKey] = origin.Credentials

	var extractRequest = exec.NewExtractRequest(source,
		exec.DefaultOptions(),
		exec.NewExtractCommand(fmt.Sprintf("svn %v --username=%v %v", command, username, strings.Join(arguments, " ")), "", nil, []string{util.NoSuchFileOrDirectory, "Event not found", "Errors validating server certificate", "Unable to connect to a repository"}),
		exec.NewExtractCommand(CredentialKey, "Password", nil, []string{util.NoSuchFileOrDirectory, "Event not found", "Username:"}),
		exec.NewExtractCommand("no", "Store password unencrypted", nil, []string{util.NoSuchFileOrDirectory, "Event not found", "Errors validating server certificate"}))
	extractRequest.TimeoutMs = 1000 * 200
	extractRequest.Terminators = append(extractRequest.Terminators, "Password for", "(yes/no)?")
	extractRequest.Secrets = secret.NewSecrets(secrets)

	if err = endly.Run(context, extractRequest, nil); err != nil {
		err = checkVersionControlAuthErrors(err, context.Secrets, origin)
	}
	if err != nil {
		return err
	}
	response, err := s.checkInfo(context, &StatusRequest{
		Source: source,
		Type:   "svn",
	})
	if response != nil {
		*info = *response.Info
	}
	return err
}

func (s *svnService) commit(context *endly.Context, request *CommitRequest) (*CommitResponse, error) {
	runResponse := &exec.RunResponse{}
	var cmd = fmt.Sprintf("svn ci -m \"%v\" ", strings.Replace(request.Message, "\"", "'", len(request.Message)))
	if err := endly.Run(context, exec.NewRunRequest(request.Source, false, cmd), runResponse); err != nil {
		return nil, err
	}
	if util.CheckNoSuchFileOrDirectory(runResponse.Stdout()) {
		return nil, fmt.Errorf("failed to commit %v", runResponse.Stdout())
	}

	var response = &CommitResponse{}
	statusResponse, err := s.checkInfo(context, &StatusRequest{
		Source: request.Source,
	})
	if err != nil {
		return nil, err
	}
	response.Info = statusResponse.Info
	return response, nil
}
