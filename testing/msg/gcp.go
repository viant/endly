package msg

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

func (s *gcPubSubClient) createSubscription(resource *ResourceSetup) (*pubsub.Subscription, error) {
	subscription, err := s.getSubscription(&resource.Resource)
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
		topic, err := s.getTopic(config.Topic)
		if err != nil {
			return nil, err
		}

		subscriptionConfig := pubsub.SubscriptionConfig{
			Labels:              config.Labels,
			Topic:               topic,
			AckDeadline:         config.AckDeadline,
			RetainAckedMessages: config.RetainAckedMessages,
			RetentionDuration:   config.RetentionDuration,
		}

		if subscription, err = s.client.CreateSubscription(s.ctx, subscription.ID(), subscriptionConfig); err != nil {
			return nil, err
		}
	}
	return subscription, err
}

func (s *gcPubSubClient) createTopic(resource *ResourceSetup) (*pubsub.Topic, error) {
	topic, err := s.getTopic(&resource.Resource)
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

func (s *gcPubSubClient) SetupResource(resource *ResourceSetup) (*Resource, error) {
	var err error
	var result = resource.Resource
	switch resource.Type {
	case ResourceTypeTopic:
		topic, err := s.createTopic(resource)
		if err != nil {
			return nil, err
		}
		result.ID = topic.ID()
	case ResourceTypeSubscription:
		subscription, err := s.createSubscription(resource)
		if err != nil {
			return nil, err
		}
		result.ID = subscription.ID()
	default:
		err = fmt.Errorf("unsupported resource type: %v, %v", resource.Type, resource.URL)
	}
	return &result, err
}

func (s *gcPubSubClient) DeleteResource(resource *Resource) error {
	switch resource.Type {
	case ResourceTypeTopic:
		topic, err := s.getTopic(resource)
		if err != nil {
			return err
		}
		return topic.Delete(s.ctx)
	case ResourceTypeSubscription:
		subscription, err := s.getSubscription(resource)
		if err != nil {
			return err
		}
		return subscription.Delete(s.ctx)
	}
	return fmt.Errorf("unsupported resource type: %v, %v", resource.Type, resource.URL)
}

func (s *gcPubSubClient) Push(dest *Resource, message *Message) (Result, error) {
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

	var pubMessage = &pubsub.Message{}
	if len(message.Attributes) > 0 {
		pubMessage.Attributes = make(map[string]string)
		for k, v := range message.Attributes {
			pubMessage.Attributes[k] = toolbox.AsString(v)
		}
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

func (s *gcPubSubClient) PullN(source *Resource, max int, nack bool) ([]*Message, error) {
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
		pulledMessage := &Message{
			ID:   msg.ID,
			Data: msg.Data,
		}
		if len(msg.Attributes) > 0 {
			pulledMessage.Attributes = make(map[string]interface{})
			for k, v := range msg.Attributes {
				pulledMessage.Attributes[k] = v
			}
		}
		messages = append(messages, pulledMessage)
		pulledCount := int(atomic.AddInt32(&pulledCounter, 1))
		if max > 0 && pulledCount >= max {
			cancel()
		}
		if nack {
			msg.Nack()
		} else {
			msg.Ack()
		}
	})
	return messages, err
}

func (s *gcPubSubClient) Close() error {
	return s.client.Close()
}

func (s *gcPubSubClient) getSubscription(dest *Resource) (*pubsub.Subscription, error) {
	if dest.Name == "" && dest.URL == "" {
		return nil, fmt.Errorf("subscription name and URL was empty, expected /projects/[PROJECT ID]/subscriptions/[SUBSCRIPTION] URL or subscription name")
	}
	if dest.projectID == "" {
		return s.client.Subscription(dest.Name), nil
	}
	return s.client.SubscriptionInProject(dest.Name, dest.projectID), nil
}

func (s *gcPubSubClient) getTopic(dest *Resource) (*pubsub.Topic, error) {
	if dest.Name == "" && dest.URL == "" {
		return nil, fmt.Errorf("subscription name and URL was empty, expected /projects/[PROJECT ID]/topics/[SUBSCRIPTION] URL or topic name")
	}
	if dest.projectID == "" {
		return s.client.Topic(dest.Name), nil
	}
	return s.client.TopicInProject(dest.Name, dest.projectID), nil
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
	var projectID = extractSubPath(URL, "project")
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
