package endly

import (
	"fmt"
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

func extractGitStatus(stdout string, response *InfoResponse) {
	if strings.Contains(stdout, "nothing to commit") {
		response.IsUptoDate = true
	}

	response.New = make([]string, 0)
	response.Modified = make([]string, 0)
	response.Deleted = make([]string, 0)
	response.Untracked = make([]string, 0)
	var state = expectChangeType

	for _, line := range strings.Split(stdout, "\r\n") {
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

func extractRevision(stdout string, response *InfoResponse) {
	if strings.Contains(stdout, "unknown revision") {
		response.IsVersionControlManaged = true
		return
	}
	response.Revision = strings.TrimSpace(stdout)

}

func (s *gitService) checkInfo(context *Context, request *StatusRequest) (*InfoResponse, error) {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	var result = &InfoResponse{}

	response, err := context.Execute(request.Target, &ManagedCommand{
		Executions: []*Execution{
			{
				Command: fmt.Sprintf("cd %v", target.ParsedURL.Path),
			},
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

	if strings.Contains(response.Stdout(1), "Not a git") {
		return result, nil
	}
	result.IsVersionControlManaged = true
	extractGitStatus(response.Stdout(1), result)
	extractRevision(response.Stdout(3), result)
	return result, nil
}

func (s *gitService) checkout(context *Context, request *CheckoutRequest) (*InfoResponse, error) {
	target, err := context.ExpandResource(request.Target)
	if err != nil {
		return nil, err
	}
	response, err := context.Execute(request.Target, &ManagedCommand{
		Executions: []*Execution{
			{
				Command: fmt.Sprintf("git clone %v %v", request.Origin.URL, target.ParsedURL.Path),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if CheckNoSuchFileOrDirectory(response.Stdout()) {
		return nil, fmt.Errorf("Failed to checkout %v", response.Stdout())
	}

	return s.checkInfo(context, &StatusRequest{
		Target: target,
	})
}

func (s *gitService) commit(context *Context, request *CommitRequest) (*InfoResponse, error) {
	checkInfo, err := s.checkInfo(context, &StatusRequest{
		Target: request.Target,
	})
	if err != nil {
		return nil, err
	}

	if len(checkInfo.Untracked) > 0 {
		for _, file := range checkInfo.Untracked {
			_, err = context.Execute(request.Target, &ManagedCommand{
				Executions: []*Execution{
					{
						Command: fmt.Sprintf("git add %v ", file),
						Error:   []string{"No such file or directory", "Error"},
					},
				},
			})
			if err != nil {
				return nil, err
			}
		}
	}

	message := strings.Replace(request.Message, "\"", "'", len(request.Message))
	_, err = context.Execute(request.Target, &ManagedCommand{
		Executions: []*Execution{
			{
				Command: fmt.Sprintf("git commit -m \"%v\" -a", message),
				Error:   []string{"No such file or directory", "Error"},
			},
		},
	})

	//TODO add branch push
	_, err = context.Execute(request.Target, &ManagedCommand{
		Executions: []*Execution{
			{
				Command: "git push",
			},
		},
	})
	return s.checkInfo(context, &StatusRequest{
		Target: request.Target,
	})
}
