package storage

import (
	"github.com/pkg/errors"
	"github.com/viant/endly"
	"github.com/viant/endly/system/cloud/gcp"
	"google.golang.org/api/storage/v1"
)

// SetNotification setup notification
func (s *service) SetNotification(context *endly.Context, request *SetupNotificationRequest) (*SetupNotificationResponse, error) {
	response := &SetupNotificationResponse{}
	return response, s.setupNotification(context, request, response)
}

func (s *service) setupNotification(context *endly.Context, request *SetupNotificationRequest, response *SetupNotificationResponse) error {
	client, err := GetClient(context)
	if err != nil {
		return err
	}
	service := storage.NewNotificationsService(client.service)
	if request.ProjectID == "" && client.CredConfig != nil {
		request.ProjectID = client.CredConfig.ProjectID
	}
	request.Topic = gcp.ExpandMeta(context, request.Topic)

	listCall := service.List(request.Bucket)
	listCall.Context(client.Context())
	listCall.UserProject(request.ProjectID)
	list, err := listCall.Do()
	if err != nil {
		return err
	}
	byTopic := make(map[string]*storage.Notification)
	for i, notification := range list.Items {
		byTopic[notification.Topic] = list.Items[i]
		deleteCall := service.Delete(request.Bucket, notification.Id)
		deleteCall.Context(client.Context())
		deleteCall.UserProject(request.ProjectID)
		if err = deleteCall.Do(); err != nil {
			return err
		}
	}
	_, ok := byTopic[request.Topic]
	if ok {
		deleteCall := service.Delete(request.Bucket, request.Notification.Id)
		deleteCall.Context(client.Context())
		deleteCall.UserProject(request.ProjectID)
	}
	insertCall := service.Insert(request.Bucket, &request.Notification)
	insertCall.Context(client.Context())
	insertCall.UserProject(request.ProjectID)
	response.Notification, err = insertCall.Do()
	if err != nil {
		err = errors.Wrapf(err, "failed to set notification bucket: %v, topic: %v", request.Bucket, request.Topic)
	}
	return err
}
