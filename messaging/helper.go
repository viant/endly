package messaging

import (
	"bytes"
	"encoding/json"
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
		return ResourceVendorGoogleCloud
	}
	return ""
}

func expandResource(context *endly.Context, resource *Resource) *Resource {
	state := context.State()
	return &Resource{
		URL:         state.ExpandAsText(resource.URL),
		Type:        state.ExpandAsText(resource.Type),
		Name:        state.ExpandAsText(resource.Name),
		Vendor:      resource.Vendor,
		Credentials: state.ExpandAsText(resource.Credentials),
	}
}
