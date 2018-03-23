package cli

import (
	"github.com/viant/endly/model"
	"strings"
)

//GetPath returns hierarchical path to the latest Activity
func GetPath(candidates *model.Activities, runner *Runner, fullPath bool) (string, int) {
	var pathLength = 0
	var activityPath = make([]string, 0)

	var activities = make([]*model.Activity, 0)
	if candidates.Len() > 0 {
		activities = append(activities, candidates.First())
	}
	if candidates.Len() > 1 {
		activities = append(activities, candidates.Last())
	}

	for i, activity := range activities {

		var tag = activity.FormatTag()
		pathLength += len(tag)
		serviceAction := ""
		if i+1 < len(activities) || fullPath {
			service := activity.Service + "."
			if activity.Service == "workflow" {
				service = ""
			}
			serviceAction = runner.ColorText(service+activity.Action, runner.ServiceActionColor)
			pathLength += len(activity.Service) + 1 + len(activity.Action)
		}

		tag = runner.ColorText(tag, runner.TagColor)
		if runner.InverseTag {
			tag = runner.ColorText(tag, "inverse")
		}

		activityPath = append(activityPath, runner.ColorText(activity.Caller, runner.PathColor)+tag+serviceAction)
		pathLength += len(activity.Caller)
	}

	var logPath = strings.Join(activityPath, runner.ColorText("|", "gray"))
	if len(activities) > 0 {
		pathLength += (len(activities) - 1)
	}
	return logPath, pathLength + 1
}
