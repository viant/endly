package cli

import (
	"strings"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"path"
	"fmt"
	"github.com/viant/toolbox/url"
)

//GetPath returns hierarchical path to the latest Activity
func GetPath(activities *endly.Activities, runner *Runner, fullPath bool) (string, int) {
	var pathLength = 0
	var activityPath = make([]string, 0)

	for i, activity := range *activities {
		var tag = activity.FormatTag()
		pathLength += len(tag)

		serviceAction := ""
		if i+1 < len(*activities) || fullPath {
			serviceAction = runner.ColorText(activity.Service+"."+activity.Action, runner.ServiceActionColor)
			pathLength += len(activity.Service) + 1 + len(activity.Action)
		}

		tag = runner.ColorText(tag, runner.TagColor)
		if runner.InverseTag {
			tag = runner.ColorText(tag, "inverse")
		}
		activityPath = append(activityPath, runner.ColorText(activity.Workflow, runner.PathColor)+tag+serviceAction)
		pathLength += len(activity.Workflow)
	}

	var path = strings.Join(activityPath, runner.ColorText("|", "gray"))
	if len(*activities) > 0 {
		pathLength += (len(*activities) - 1)
	}
	return path, pathLength + 1
}



//LoadRunRequestWithOption load WorkflowRun request and runner options
func LoadRunRequestWithOption(workflowRunRequestURL string, params ...interface{}) (*endly.RunRequest, *RunnerReportingOptions, error) {
	request := &endly.RunRequest{}
	resource := url.NewResource(workflowRunRequestURL)
	parametersMap := toolbox.Pairs(params...)
	err := resource.JSONDecode(request)
	if err != nil {
		return nil, nil, err
	}
	if len(request.Params) == 0 {
		request.Params = parametersMap
	}
	for k, v := range parametersMap {
		request.Params[k] = v
	}
	options := &RunnerReportingOptions{}
	_ = resource.JSONDecode(options)
	if options.Filter == nil {
		options.Filter = DefaultRunnerReportingOption().Filter
	}
	return request, options, nil
}

func getWorkflowURL(candidate string) (string, string, error) {
	var _, name = path.Split(candidate)
	if path.Ext(candidate) == "" {
		candidate = candidate + ".csv"
	} else {
		name = string(name[:len(name)-4]) //remove extension
	}
	resource := url.NewResource(candidate)
	if _, err := resource.Download(); err != nil {
		resource = url.NewResource(fmt.Sprintf("mem://%v/workflow/%v", endly.Namespace, candidate))
		if _, memError := resource.Download(); memError != nil {
			return "", "", err
		}
	}
	return name, resource.URL, nil
}
