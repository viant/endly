package msg

import (
	"context"
	"github.com/pkg/errors"
	"github.com/segmentio/kafka-go"
	"github.com/viant/toolbox"
	"strings"
	"time"
)

const keyAttribute = "key"
const idAttribute = "id"

type kafkaClient struct {
	timeout time.Duration
}

func (k *kafkaClient) Push(ctx context.Context, dest *Resource, message *Message) (Result, error) {
	config := kafka.WriterConfig{
		Brokers:      dest.Brokers,
		Topic:        dest.Name,
		Balancer:     &kafka.LeastBytes{},
	}
	body := toolbox.AsString(message.Data)
	writer := kafka.NewWriter(config)
	key := ""

	for k := range message.Attributes {
		candidate := strings.ToLower(k)
		if candidate == keyAttribute || candidate == idAttribute {
			key = toolbox.AsString(message.Attributes[k])
		}
	}
	messages := make([]kafka.Message, 0)
	messages = append(messages, kafka.Message{
		Partition: dest.Partition,
		Key:       []byte(key),
		Value:     []byte(body),
	})
	err := writer.WriteMessages(ctx, messages...)
	if err != nil {
		return nil, err
	}
	_ = writer.Close()
	return key, nil
}

func (k *kafkaClient) PullN(ctx context.Context, source *Resource, count int, nack bool) ([]*Message, error) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   source.Brokers,
		Topic:     source.Name,
		Partition: source.Partition,
		MinBytes:  10e3, // 10KB
		MaxBytes:  10e6, // 10MB
		MaxWait:   k.timeout,
	})
	if source.Offset > 0 {
		if err := reader.SetOffset(int64(source.Offset)); err != nil {
			return nil, errors.Wrapf(err, "failed to set offset: %v", source.Offset)
		}
	}
	var result = make([]*Message, 0)
	for i := 0; i < count; i++ {
		message, err := reader.ReadMessage(ctx)
		if err != nil {
			return nil, err
		}
		msg := &Message{
			Data:       message.Value,
			Attributes: map[string]interface{}{},
		}
		if len(message.Key) > 0 {
			msg.Attributes[keyAttribute] = string(message.Key)
		}
		result = append(result, msg)
		if ! nack {
			if err = reader.CommitMessages(ctx, message); err != nil {
				return nil, errors.Wrapf(err, "failed to commit message: %v", msg)
			}
		}
	}
	return result, nil
}

func (k *kafkaClient) SetupResource(resource *ResourceSetup) (*Resource, error) {
	conn, err := kafka.DialLeader(context.Background(), "tcp", resource.Brokers[0], resource.Name, resource.Partition)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to %v", resource.Brokers[0])
	}
	if resource.Recreate {
		_ = conn.DeleteTopics(resource.Name)
	}
	topicConfig := kafka.TopicConfig{
		Topic:         resource.Name,
		ReplicationFactor:resource.ReplicationFactor,
		NumPartitions: resource.Partitions,
	}
	return &resource.Resource, conn.CreateTopics(topicConfig)
}

func (k *kafkaClient) DeleteResource(resource *Resource) error {
	conn, err := kafka.DialLeader(context.Background(), "tcp", resource.Brokers[0], resource.Name, resource.Partition)
	if err != nil {
		return errors.Wrapf(err, "failed to connect to %v", resource.Brokers[0])
	}
	return conn.DeleteTopics(resource.Name)
}

func (k *kafkaClient) Close() error {
	return nil
}

func newKafkaClient(timeout time.Duration) (Client, error) {
	return &kafkaClient{timeout: timeout}, nil
}
