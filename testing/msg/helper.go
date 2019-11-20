package msg

import (
	"bytes"
	"encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/cred"
	"strings"
)

func loadMessages(data []byte) []*Message {
	var result = make([]*Message, 0)
	var text = string(data)

	if toolbox.IsNewLineDelimitedJSON(text) {
		if records, err := toolbox.NewLineDelimitedJSON(text); err == nil {
			for _, record := range records {
				if recordMap, ok := record.(map[string]interface{}); ok {
					msg := &Message{}
					err := toolbox.DefaultConverter.AssignConverted(msg, recordMap)
					if err != nil || (msg.Data == nil && len(msg.Attributes) == 0) {
						msg.Data = recordMap
					}
					result = append(result, msg)
				}
			}
			return result
		}
	}
	err := json.NewDecoder(bytes.NewReader(data)).Decode(&result)
	if err != nil {
		for _, line := range strings.Split(text, "\n") {
			msg := &Message{}
			msg.Data = line
			result = append(result, msg)
		}
	}
	return result
}

//extractSubPath extract a next matched path fragment i.e iPath /proejcts/x/topics/t1,  returns t1 for 'topics' match
func extractSubPath(aPath, match string) string {
	fragments := strings.Split(aPath, "/")
	for i := 0; i < len(fragments)-1; i++ {
		if strings.Contains(fragments[i], match) {
			return fragments[i+1]
		}
	}
	return ""
}

func inferResourceTypeFromCredentialConfig(credConfig *cred.Config) string {
	if credConfig.Key != "" && credConfig.Secret != "" {
		return ResourceVendorAmazonWebService
	} else if credConfig.ProjectID != "" {
		return ResourceVendorGoogleCloudPlatform
	}
	return ""
}

func expandResource(context *endly.Context, resource *Resource) *Resource {
	state := context.State()
	return &Resource{
		URL:               state.ExpandAsText(resource.URL),
		Type:              state.ExpandAsText(resource.Type),
		Name:              state.ExpandAsText(resource.Name),
		Vendor:            resource.Vendor,
		Credentials:       state.ExpandAsText(resource.Credentials),
		Brokers:           resource.Brokers,
		Partitions:        resource.Partitions,
		Partition:         resource.Partition,
		Offset:            resource.Offset,
		ReplicationFactor: resource.ReplicationFactor,
	}
}

func getAttributeDataType(value interface{}) string {
	dataType := "String"
	if toolbox.IsInt(value) || toolbox.IsFloat(value) {
		dataType = "Number"
	}
	return dataType
}

func putSqsMessageAttributes(attributes map[string]interface{}, target map[string]*sqs.MessageAttributeValue) {
	for k, v := range attributes {
		if v == nil {
			continue
		}
		dataType := getAttributeDataType(v)
		target[k] = &sqs.MessageAttributeValue{
			DataType:    &dataType,
			StringValue: aws.String(toolbox.AsString(v)),
		}
	}
}

func putSnsMessageAttributes(attributes map[string]interface{}, target map[string]*sns.MessageAttributeValue) {
	for k, v := range attributes {
		if v == nil {
			continue
		}
		dataType := getAttributeDataType(v)
		target[k] = &sns.MessageAttributeValue{
			DataType:    &dataType,
			StringValue: aws.String(toolbox.AsString(v)),
		}
	}
}
