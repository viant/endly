package pubsub

import (
	"cloud.google.com/go/pubsub"
	"context"
	"fmt"
	"github.com/viant/toolbox"
	"github.com/viant/toolbox/cred"
	context2 "golang.org/x/net/context"
	"google.golang.org/api/option"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type gcPubSubClient struct {
	ctx       context.Context
	client    *pubsub.Client
	projectId string
	timeout   time.Duration
}

func (s *gcPubSubClient) createSubscription(resource *Resource) (*pubsub.Subscription, error) {
	subscription, err := s.getSubscription(resource.URL)
	if err != nil {
		return nil, err
	}
	exists, err := subscription.Exists(s.ctx)
	if err != nil {
		return nil, err
	}
	if exists && resource.Recreate {
		exists = false
		if err = subscription.Delete(s.ctx); err != nil {
			return nil, err
		}
	}
	if !exists {
		config := resource.Config
		if config.Topic == nil {
			return nil, fmt.Errorf("topic was empty")
		}
		topicURL := getTopicURL(resource)
		topic, err := s.getTopic(topicURL)
		if err != nil {
			return nil, err
		}

		sbuscriptionConfig := pubsub.SubscriptionConfig{
			Labels:              config.Labels,
			Topic:               topic,
			AckDeadline:         config.AckDeadline,
			RetainAckedMessages: config.RetainAckedMessages,
			RetentionDuration:   config.RetentionDuration,
		}

		subscription, err = s.client.CreateSubscription(s.ctx, subscription.ID(), sbuscriptionConfig)
	}
	return subscription, err
}

func (s *gcPubSubClient) createTopic(resource *Resource) (*pubsub.Topic, error) {
	topic, err := s.getTopic(resource.URL)
	if err != nil {
		return nil, err
	}
	exists, err := topic.Exists(s.ctx)
	if err != nil {
		return nil, err
	}
	if exists && resource.Recreate {
		exists = false
		if err = topic.Delete(s.ctx); err != nil {
			return nil, err
		}
	}

	if !exists {
		if topic, err = s.client.CreateTopic(s.ctx, topic.ID()); err != nil {
			return nil, err
		}
	}
	return topic, nil
}

func (s *gcPubSubClient) Create(resource *Resource) (*Resource, error) {
	var err error
	switch resource.Type {
	case ResourceTypeTopic:
		_, err = s.createTopic(resource)
	case ResourceTypeSubscription:
		_, err = s.createSubscription(resource)
	default:
		err = fmt.Errorf("unsupported resource type: %v, %v", resource.Type, resource.URL)
	}
	return resource, err
}

func (s *gcPubSubClient) Delete(resource *Resource) error {
	switch resource.Type {
	case ResourceTypeTopic:
		topic, err := s.getTopic(resource.URL)
		if err != nil {
			return err
		}
		return topic.Delete(s.ctx)
	case ResourceTypeSubscription:
		subscription, err := s.getSubscription(resource.URL)
		if err != nil {
			return err
		}
		return subscription.Delete(s.ctx)
	}
	return fmt.Errorf("unsupported resource type: %v, %v", resource.Type, resource.URL)
}

func (s *gcPubSubClient) Push(dest string, message *Message) (Result, error) {
	topic, err := s.getTopic(dest)
	if err != nil {
		return nil, err
	}
	ok, err := topic.Exists(s.ctx)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("topic %v does not exist", dest)
	}
	var pubMessage = &pubsub.Message{
		Attributes: message.Attributes,
	}
	if message.Data != nil {
		switch data := message.Data.(type) {
		case string:
			pubMessage.Data = []byte(data)
		case []byte:
			pubMessage.Data = data
		default:
			JSONText, err := toolbox.AsIndentJSONText(data)
			if err != nil {
				return nil, err
			}
			pubMessage.Data = []byte(JSONText)
		}
	}
	response := topic.Publish(s.ctx, pubMessage)
	serverId, err := response.Get(s.ctx)
	if err != nil {
		return response, err
	}
	select {
	case <-response.Ready():
	case <-time.After(s.timeout):
		log.Printf("publish ready timeout reached: %s", s.timeout)
	}
	return serverId, err
}

func (s *gcPubSubClient) PullN(source string, max int) ([]*Message, error) {
	subscription, err := s.getSubscription(source)
	if err != nil {
		return nil, err
	}
	var pulledCounter int32 = 0
	ctx, cancel := context.WithCancel(s.ctx)
	go func() {
		time.Sleep(s.timeout)

		pulledCount := int(atomic.AddInt32(&pulledCounter, 1))
		if pulledCount != max && max > 0 {
			log.Printf("publish ready timeout reached: %s", s.timeout)
			cancel()
		}
	}()
	mutex := &sync.Mutex{}
	var messages = make([]*Message, 0)
	err = subscription.Receive(ctx, func(ctx context2.Context, msg *pubsub.Message) {
		mutex.Lock()
		defer mutex.Unlock()
		messages = append(messages, &Message{
			ID:         msg.ID,
			Attributes: msg.Attributes,
			Data:       msg.Data,
		})
		pulledCount := int(atomic.AddInt32(&pulledCounter, 1))
		if max > 0 && pulledCount >= max {
			cancel()
		}
		msg.Ack()
	})
	return messages, err
}

func (s *gcPubSubClient) Close() error {
	return s.client.Close()
}

func (s *gcPubSubClient) getSubscription(dest string) (*pubsub.Subscription, error) {
	projectId, subscription, err := extractGCSubcription(dest)
	if err != nil {
		return nil, err
	}
	if projectId == "" {
		projectId = s.projectId
	}
	return s.client.SubscriptionInProject(subscription, projectId), nil
}

func (s *gcPubSubClient) getTopic(dest string) (*pubsub.Topic, error) {
	projectId, topic, err := extractGCTopics(dest)
	if err != nil {
		return nil, err
	}
	if projectId == "" {
		projectId = s.projectId
	}
	return s.client.TopicInProject(topic, projectId), nil
}

func newCloudPubSub(credConfig *cred.Config, URL string, timeout time.Duration) (Client, error) {
	ctx := context.Background()
	jwtConfig, err := credConfig.NewJWTConfig(pubsub.ScopePubSub)
	if err != nil {
		return nil, err
	}
	var opts = []option.ClientOption{
		option.WithTokenSource(jwtConfig.TokenSource(ctx)),
	}
	client, err := pubsub.NewClient(ctx, credConfig.ProjectID, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub client: %v", err)
	}
	var projectID = extractGCProjectID(URL)
	if projectID == "" {
		projectID = credConfig.ProjectID
	}

	var service = &gcPubSubClient{
		client:    client,
		ctx:       ctx,
		timeout:   timeout,
		projectId: projectID,
	}
	return service, nil
}
