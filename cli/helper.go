package cli

import (
	"github.com/viant/endly/model"
	"strings"
)

// GetPath returns hierarchical path to the latest Activity
func GetPath(candidates *model.Activities, runner *Runner, fullPath bool) string {
	var activityPath = make([]string, 0)
	var activities = make([]*model.Activity, 0)
	for i := 0; i < candidates.Len(); i++ {
		if i > 0 && candidates.Get(i).Service == candidates.Get(i-1).Service {
			continue
		}
		activities = append(activities, candidates.Get(i))
	}

	if len(activities) > 2 {
		activities = activities[len(activities)-2:]
	}

	for i, activity := range activities {
		var tag = activity.FormatTag()
		serviceAction := ""
		if i+1 < len(activities) || fullPath {
			service := activity.Service + "."
			if activity.Service == "workflow" {
				service = ""
			}
			serviceAction = runner.ColorText(service+activity.Action, runner.ServiceActionColor)
		}
		tag = runner.ColorText(tag, runner.TagColor)
		if runner.InverseTag {
			tag = runner.ColorText(tag, "inverse")
		}

		activityPath = append(activityPath, runner.ColorText(activity.Caller, runner.PathColor)+tag+serviceAction)
	}
	var resut = strings.Join(activityPath, runner.ColorText("|", "gray"))
	return resut
}
