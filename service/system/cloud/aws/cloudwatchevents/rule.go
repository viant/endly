package cloudwatchevents

import (
	"github.com/aws/aws-sdk-go/service/cloudwatchevents"
)

func hasRuleChanged(rule *cloudwatchevents.Rule, input *cloudwatchevents.PutRuleInput) bool {
	if rule == nil {
		return true
	}
	if input.State != nil && !isStringPtrEqual(input.State, rule.State) {
		return true
	}
	if input.Description != nil && !isStringPtrEqual(input.Description, rule.Description) {
		return true
	}
	if input.EventPattern != nil && !isStringPtrEqual(input.EventPattern, rule.EventPattern) {
		return true
	}
	if input.ScheduleExpression != nil && !isStringPtrEqual(input.ScheduleExpression, rule.ScheduleExpression) {
		return true
	}
	if input.RoleArn != nil && !isStringPtrEqual(input.RoleArn, rule.RoleArn) {
		return true
	}
	return false
}
