package s3

import (
	"github.com/aws/aws-sdk-go/service/s3"
)

type NotificationFilter struct {
	Prefix []string
	Suffix []string
}


func (f *NotificationFilter) ToNotificationConfigurationFilter() *s3.NotificationConfigurationFilter{
	if f == nil || len(f.Suffix) + len(f.Prefix) == 0 {
		return nil
	}
	result := &s3.NotificationConfigurationFilter{
		Key:&s3.KeyFilter{
			FilterRules:make([]*s3.FilterRule, 0),
		},
	}
	appendRules(f.Prefix, "prefix", &result.Key.FilterRules)
	appendRules(f.Suffix, "suffix", &result.Key.FilterRules)
	return result
}


func appendRules(values []string, ruleName string, rules *[]*s3.FilterRule)  {
	if len(values) == 0 {
		return
	}
	for _, value := range values {
		*rules = append(*rules,  &s3.FilterRule{
			Name: &ruleName,
			Value:&value,
		})
	}

}