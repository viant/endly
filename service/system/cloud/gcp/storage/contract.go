package storage

import (
	"errors"
	"fmt"
	"google.golang.org/api/storage/v1"
	"strings"
)

// SetNotification represents setup notification request
type SetupNotificationRequest struct {
	Bucket    string
	ProjectID string
	storage.Notification
}

// SetNotification represents setup notification response
type SetupNotificationResponse struct {
	*storage.Notification
}

// Init initialises request
func (r *SetupNotificationRequest) Init() error {
	elements := strings.Split(r.Topic, "/")
	if len(elements) == 1 {
		r.Topic = fmt.Sprintf("//pubsub.googleapis.com/projects/${gcp.projectId}/topics/%v", r.Topic)
	} else if r.ProjectID == "" && len(elements) > 3 {
		r.ProjectID = elements[len(elements)-3]
	}
	if r.PayloadFormat == "" {
		r.PayloadFormat = "NONE"
	}
	return nil
}

// Validate checks if request is valid
func (r *SetupNotificationRequest) Validate() error {
	if r.Bucket == "" {
		return errors.New("bucket was empty")
	}
	if r.Topic == "" {
		return errors.New("topic was empty")
	}
	return nil
}
