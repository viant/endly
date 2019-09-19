package lambda

import (
	"encoding/json"
	"fmt"
	acloudwatchevents "github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"github.com/pkg/errors"
	"github.com/viant/endly/system/cloud/aws/cloudwatchevents"
)


var defaultEventSource = "com.company.app"
var defaultEventType = "appRequestSubmitted"

type ScheduleEvent struct {
	Detail     interface{}
	Source     *string
	DetailType *string
}

//Schedule represents schedule
type Schedule struct {
	Expression  *string
	Event *ScheduleEvent
}


func (e *ScheduleEvent) Init() error {
	if e.Source == nil {
		e.Source = &defaultEventSource
	}
	if e.DetailType == nil {
		e.DetailType = &defaultEventType
	}
	if e.Detail == nil {
		var detail = ""
		e.Detail =  &detail
	}

	return nil
}


func (s *Schedule) Init() error {
	if s.Event == nil {
		s.Event = &ScheduleEvent{}
	}
	return s.Event.Init()
}

//ScheduleDeployRule returns DeployRuleInput
func (i *DeployInput) ScheduleDeployRule() *cloudwatchevents.DeployRuleInput {
	if i.Schedule == nil || i.Schedule.Expression == nil {
		return nil
	}
	input := &cloudwatchevents.DeployRuleInput{
		Targets: make([]*cloudwatchevents.Target, 0),
	}
	input.ScheduleExpression = i.Schedule.Expression
	name := fmt.Sprintf("%vScheduler", *i.FunctionName)
	input.Name = &name
	roleName := fmt.Sprintf("%vScheduler", *i.RoleName)
	input.RoleName = &roleName
	input.Targets = append(input.Targets, &cloudwatchevents.Target{
		Function: i.FunctionName,
	})
	return input
}

func (i *DeployInput) ScheduleEventsInput(resourceARN *string) (*acloudwatchevents.PutEventsInput, error) {
	input := &acloudwatchevents.PutEventsInput{
		Entries: make([]*acloudwatchevents.PutEventsRequestEntry, 0),
	}
	if resourceARN == nil {
		return  nil, fmt.Errorf("schedule rule arn was empty")
	}
	schedule := i.Schedule
	event := schedule.Event
	detail, ok := event.Detail.(string)
	if !ok && event.Detail != nil {
		JSON, err := json.Marshal(event.Detail)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal	 event detail: %v", event.Detail)
		}
		detail = string(JSON)
	}

	input.Entries = append(input.Entries, &acloudwatchevents.PutEventsRequestEntry{
		Detail: &detail,
		Source: event.Source,
		DetailType:event.DetailType,
		Resources: []*string{resourceARN},
	})
	return input, nil

}
