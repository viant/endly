package vc

import (
	"fmt"
	"github.com/viant/endly"
	"path"
	"strings"
)

type svnService struct{}

func (s *svnService) checkInfo(context *endly.Context, request *StatusRequest) (*InfoResponse, error) {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	var result = &InfoResponse{}

	response, err := context.Execute(request.Target, &endly.ManagedCommand{
		Executions: []*endly.Execution{
			{
				Command: fmt.Sprintf("cd %v", target.ParsedURL.Path),
			},
			{
				Command: fmt.Sprintf("svn info"),
				Extraction: []*endly.DataExtraction{{
					RegExpr: "URL: ([^\\s]+)",
					Name:    "origin",
				},
					{
						RegExpr: "Revision: ([^\\s]+)",
						Name:    "revision",
					}},
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
	if strings.Contains(response.Stdout[1], "is not a working copy") {
		return result, nil
	}
	result.IsVersionControlManaged = true

	readSvnStatus(response, result)
	return result, nil
}

func readSvnStatus(commandResult *endly.CommandInfo, response *InfoResponse) {
	response.New = make([]string, 0)
	response.Modified = make([]string, 0)
	response.Deleted = make([]string, 0)
	response.Untracked = make([]string, 0)
	for _, line := range strings.Split(commandResult.Stdout[2], "\r\n") {
		if len(line) == 0 {
			continue
		}
		fileStatus := string(line[0:1])
		file := strings.Trim(string(line[1:]), " \t")
		switch fileStatus {
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

func (s *svnService) checkout(context *endly.Context, request *CheckoutRequest) (*InfoResponse, error) {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}

	username, password, err := request.Origin.LoadCredential()
	if err != nil {
		return nil, err
	}

	_, err = context.Execute(request.Target, &endly.ManagedCommand{
		Options: &endly.ExecutionOptions{
			TimeoutMs:   1000 * 30,
			Terminators: []string{"Username", "Password for", "(yes/no)?", "Checked out revision"},
		},
		Executions: []*endly.Execution{
			{
				Command: fmt.Sprintf("svn co --username=%v %v  %v", username, request.Origin.URL, target.ParsedURL.Path),
				Error:   []string{"No such file or directory", "event not found"},
			},
			{
				MatchOutput: "Password for",
				Command:     fmt.Sprintf("%v", password),
				Error:       []string{"No such file or directory", "event not found"},
			},
			{
				MatchOutput: "Store password unencrypted",
				Command:     "no",
				Error:       []string{"No such file or directory", "event not found"},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return s.checkInfo(context, &StatusRequest{
		Target: target,
	})
}

func (s *svnService) commit(context *endly.Context, request *CommitRequest) (*InfoResponse, error) {

	_, err := context.Execute(request.Target, &endly.ManagedCommand{
		Executions: []*endly.Execution{
			{
				Command: fmt.Sprintf("svn ci -m \"%v\" ", strings.Replace(request.Message, "\"", "'", len(request.Message))),
				Error:   []string{"No such file or directory", "Error"},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return s.checkInfo(context, &StatusRequest{
		Target: request.Target,
	})
}
