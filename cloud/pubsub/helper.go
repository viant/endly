package pubsub

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/viant/toolbox"
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

func extractGCTopics(URI string) (string, string, error) {
	var projectId, topic string
	if strings.Contains(URI, "/topics/") {
		var fragments = strings.Split(URI, "/")
		for i := 0; i < len(fragments)-1; i++ {
			if projectId == "" && fragments[i] == "projects" {
				projectId = fragments[i+1]
				i++
			}
			if topic == "" && fragments[i] == "topics" {
				topic = fragments[i+1]
				i++
			}
		}
	} else {
		index := strings.LastIndex(URI, ":")
		if index == -1 || strings.Count(URI, "/") > 2 {
			return "", "", fmt.Errorf("invalid URL expected: gcpubsub:/projects/[PROJECT ID]/topics/[TOPIC] or gcpubsub:/[TOPIC] but had: %s", URI)
		}
		topic = strings.Trim(string(URI[index+1:]), "/")
	}
	return projectId, topic, nil
}

func extractGCSubcription(URI string) (string, string, error) {
	var projectId, subscription string
	if strings.Contains(URI, "/subscriptions/") {
		var fragments = strings.Split(URI, "/")
		for i := 0; i < len(fragments)-1; i++ {
			if projectId == "" && fragments[i] == "projects" {
				projectId = fragments[i+1]
				i++
			}
			if subscription == "" && fragments[i] == "subscriptions" {
				subscription = fragments[i+1]
				i++
			}
		}
	} else {
		index := strings.LastIndex(URI, ":")
		if index == -1 || strings.Count(URI, "/") > 2 {
			return "", "", fmt.Errorf("invalid URL expected: gcpubsub:/projects/[PROJECT ID]/subscriptions/[SUBSCRIPTION] or gcpubsub:/[SUBSCRIPTION] but had: %s", URI)
		}
		subscription = strings.Trim(string(URI[index+1:]), "/")
	}
	return projectId, subscription, nil
}

func extractGCProjectID(URI string) string {
	if strings.Contains(URI, "/projects/") {
		var fragments = strings.Split(URI, "/")
		for i := 0; i < len(fragments)-1; i++ {
			if fragments[i] == "projects" {
				return fragments[i+1]
			}
		}
	}
	return ""
}

func getTopicURL(resource *Resource) string {
	if resource.Config == nil || resource.Config.Topic == nil {
		return ""
	}

	resource.Init()
	scheme := resource.ParsedURL.Scheme
	topicURL := resource.Config.Topic.URL
	if !strings.HasPrefix(topicURL, scheme) {
		return scheme + ":" + resource.Config.Topic.URL
	}
	return topicURL
}
