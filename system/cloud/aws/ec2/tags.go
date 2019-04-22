package ec2

import "github.com/aws/aws-sdk-go/service/ec2"

func matchesByTags(expect map[string]string, actual []*ec2.Tag) bool {
	if len(expect) > len(actual) || len(expect) == 0 {
		return false
	}
	actualTags := indexTags(actual)
	for key, expecValue:= range expect  {
		actualValue, ok := actualTags[key]
		if ! ok || expecValue != actualValue {
			return false
		}
	}
	return true
}


func indexTags(tags []*ec2.Tag) map[string]string {
	var result = make(map[string]string)
	if len(tags) == 0 {
		return result
	}
	for _, tag := range tags {
		if tag.Key == nil {
			continue
		}
		value := ""
		if tag.Value != nil {
			value = *tag.Value
		}
		result[*tag.Key] = value
	}
	return result
}
