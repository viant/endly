package msg

import (
	"fmt"
	"strings"
	"time"
)

const (
	ResourceTypeTopic        = "topic"
	ResourceTypeSubscription = "subscription"
	ResourceTypeQueue        = "queue"
)

type Resource struct {
	URL         string
	Credentials string
	ID          string
	Name        string
	Type        string `description:"resource type: topic, subscription"`
	Vendor      string
	Config      interface{} `description:"vendor client config"`
	projectID   string
}

//Init initializes resource
func (r *Resource) Init() error {
	if r == nil {
		return fmt.Errorf("resource was empty")
	}
	if r.URL != "" {
		r.projectID = extractSubPath(r.URL, "project")
		if r.Name == "" {
			r.Name = r.URL
			index := strings.LastIndex(r.URL, "/")
			if index == -1 {
				index = strings.LastIndex(r.URL, ":")
			}
			if index != -1 {
				r.Name = string(r.URL[index+1:])
			}
		}
	}
	return nil
}

//NewResource creates a new resource
func NewResource(resourceType, URL, credentials string) *Resource {
	return &Resource{
		Type:        resourceType,
		URL:         URL,
		Credentials: credentials,
	}
}

//Resource represents resource setup
type ResourceSetup struct {
	Resource
	Recreate bool
	Config   *Config
}

//Init initializes setup resource
func (r *ResourceSetup) Init() error {
	if r.Type == "" {
		if isTopic := r.Config == nil || r.Config.Topic == nil; isTopic {
			r.Type = ResourceTypeTopic
		} else {
			r.Type = ResourceTypeSubscription
			r.Config.Topic.Type = ResourceTypeTopic
		}
	}

	if r.Config != nil && r.Config.Topic != nil {
		_ = r.Config.Topic.Init()
	}
	return r.Resource.Init()
}

func (r *ResourceSetup) Validate() error {
	if r.Type == ResourceTypeSubscription && (r.Vendor == ResourceVendorGoogleCloud || r.Vendor == ResourceVendorGoogleCloudPlatform) {
		if r.Config == nil {
			return fmt.Errorf("subscription config was empty")
		}
		if r.Config.Topic == nil {
			return fmt.Errorf("subscription config.Topic was empty")
		}
	}
	if r.Type == ResourceTypeQueue {
		if r.Name == "" {
			return fmt.Errorf("Name was empty")
		}
	}

	return nil
}

//NewResourceSetup creates a new URL
func NewResourceSetup(resourceType, URL, credentials string, recreate bool, config *Config) *ResourceSetup {
	return &ResourceSetup{
		Resource: Resource{
			Type:        resourceType,
			Credentials: credentials,
			URL:         URL,
		},
		Recreate: recreate,
		Config:   config,
	}
}

//Config represent a subscription config
type Config struct {
	Topic               *Resource
	Labels              map[string]string
	Attributes          map[string]string
	AckDeadline         time.Duration
	RetentionDuration   time.Duration
	RetainAckedMessages bool
}

//NewConfig create new config
func NewConfig(topic string) *Config {
	return &Config{
		Topic: &Resource{Name: topic, Type: ResourceTypeTopic},
	}
}
