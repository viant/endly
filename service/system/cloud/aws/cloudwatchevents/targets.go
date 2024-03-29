package cloudwatchevents

import "github.com/aws/aws-sdk-go/service/cloudwatchevents"

// Target represents target
type Target struct {
	cloudwatchevents.Target
	Function *string `description:"lambda function name"`
}

// Target represents targets
type Targets []*Target

// Targets returns targets
func (t Targets) targets() []*cloudwatchevents.Target {
	var result = make([]*cloudwatchevents.Target, len(t))
	for i := range t {
		result[i] = &t[i].Target
	}
	return result
}

func (t Targets) hasChanged(targets []*cloudwatchevents.Target) bool {
	if len(t) != len(targets) {
		return true
	}
	dest := make(map[string]*cloudwatchevents.Target)
	source := make(map[string]*cloudwatchevents.Target)
	for i := range t {
		dest[*t[i].Arn] = &t[i].Target
	}
	for i := range targets {
		source[*targets[i].Arn] = targets[i]
	}
	for k := range dest {
		src, ok := source[k]
		if !ok {
			return true
		}
		if hasTargerChanged(src, dest[k]) {
			return true
		}
	}

	return false

}

func isStringPtrEqual(val1, val2 *string) bool {
	if val1 == nil || val2 == nil {
		return val2 == val1
	}
	return *val2 == *val1
}

func hasTargerChanged(source, dest *cloudwatchevents.Target) bool {
	if !isStringPtrEqual(dest.RoleArn, source.RoleArn) {
		return true
	}
	if !isStringPtrEqual(dest.Input, source.Input) {
		return true
	}
	if !isStringPtrEqual(dest.InputPath, source.InputPath) {
		return true
	}
	return false
}
