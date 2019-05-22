package slack

import (
	"encoding/json"
	"fmt"
	"github.com/go-errors/errors"
	"github.com/viant/endly/testing/validator"
	"gopkg.in/yaml.v2"
	"strings"
)

const PullDefaultTimeoutMs = 30000

//PostRequest represents post request
type PostRequest struct {
	Credentials string
	Channel     string
	Messages    []*Message
}

//PostResponse represnets a post response
type PostResponse struct {
	Channel   string
	Timestamp string
}

//ListenRequest represents a listen request
type ListenRequest struct {
	Credentials string
	Channels    []string
	channels    map[string]bool
}

//ListenResponse represents listen response
type ListenResponse struct{}

//AssertRequest represents assert request
type PullRequest struct {
	Expect    []*Message
	TimeoutMs int
	Count     int
}

//AssertResponse represents assert response
type PullResponse struct {
	Messages []*Message
	Assert   *validator.AssertResponse
}

//Init initializes request
func (r *PostRequest) Init() error {
	if r.Channel != "" && len(r.Messages) > 0 {
		for _, message := range r.Messages {
			if message.Channel == "" {
				message.Channel = r.Channel
			}

			if message.Asset != nil && message.Asset.Data != nil && message.Asset.Content == "" {
				switch strings.ToLower(message.Asset.Type) {
				case "json":
					payload, err := json.Marshal(message.Asset.Data)
					if err != nil {
						return err
					}
					message.Asset.Content = string(payload)
				case "yaml":
					payload, err := yaml.Marshal(message.Asset.Data)
					if err != nil {
						return err
					}
					message.Asset.Content = string(payload)
				}
			}
		}
	}
	return nil
}

//Validate check if a request is valid
func (r *PostRequest) Validate() error {
	if r.Credentials == "" {
		return errors.New("credentials were empty")
	}
	if len(r.Messages) == 0 {
		return fmt.Errorf("message was empty")
	}

	for i, message := range r.Messages {
		if message.Asset == nil && message.Text == "" {
			return fmt.Errorf("messages[%d].text was empty", i)
		}
		if message.Channel == "" {
			return fmt.Errorf("messages[%d].channel was empty", i)
		}
	}
	return nil
}

//Init init a request
func (r *ListenRequest) Init() error {
	r.channels = make(map[string]bool)
	if len(r.Channels) > 0 {
		for _, channel := range r.Channels {
			r.channels[channel] = true
		}
	}
	return nil
}

//Validate check if a request is valid
func (r *ListenRequest) Validate() error {
	if r.Credentials == "" {
		return errors.New("credentials were empty")
	}

	return nil
}

//Init initializes request
func (r *PullRequest) Init() error {
	if r.Count == 0 {
		r.Count = len(r.Expect)
	}
	if r.TimeoutMs == 0 {
		r.TimeoutMs = PullDefaultTimeoutMs
	}
	if r.Count == 0 {
		r.Count = 1
	}
	return nil
}
