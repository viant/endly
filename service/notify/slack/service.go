package slack

import (
	"bytes"
	"fmt"
	"github.com/nlopes/slack"
	"github.com/viant/endly"
	"github.com/viant/endly/service/testing/validator"
	"log"
	"strings"
	"sync"
	"time"
)

const (
	//ServiceID represents a data store unit service id
	ServiceID = "slack"
)

type service struct {
	*endly.AbstractService
}

func (s *service) getMessages() *Messages {
	s.Lock()
	defer s.Unlock()
	state := s.State()
	_, ok := state.GetValue(messagesKey)
	if !ok {
		state.Put(messagesKey, NewMessages())
	}
	messages, _ := state.GetValue(messagesKey)
	return messages.(*Messages)
}

func (s *service) registerRoutes() {

	s.Register(&endly.Route{
		Action: "post",
		RequestInfo: &endly.ActionInfo{
			Description: "post message to slack channel",
		},
		RequestProvider: func() interface{} {
			return &PostRequest{}
		},
		ResponseProvider: func() interface{} {
			return &PostResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*PostRequest); ok {
				return s.post(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "listen",
		RequestInfo: &endly.ActionInfo{
			Description: "listen for slack messages",
		},
		RequestProvider: func() interface{} {
			return &ListenRequest{}
		},
		ResponseProvider: func() interface{} {
			return &ListenResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*ListenRequest); ok {
				return s.listen(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})

	s.Register(&endly.Route{
		Action: "pull",
		RequestInfo: &endly.ActionInfo{
			Description: "pull/validate slack messages",
		},
		RequestProvider: func() interface{} {
			return &PullRequest{}
		},
		ResponseProvider: func() interface{} {
			return &PullResponse{}
		},
		Handler: func(context *endly.Context, request interface{}) (interface{}, error) {
			if req, ok := request.(*PullRequest); ok {
				return s.pull(context, req)
			}
			return nil, fmt.Errorf("unsupported request type: %T", request)
		},
	})
}

func (s *service) post(context *endly.Context, request *PostRequest) (*PostResponse, error) {
	client, err := getClient(context, request.Credentials)
	if err != nil {
		return nil, err
	}
	response := &PostResponse{}
	for _, message := range request.Messages {
		channelID, err := getChannelId(message.Channel, client)
		if err != nil {
			return nil, err
		}
		if message.Text != "" {
			if response.Channel, response.Timestamp, err = client.PostMessage(channelID, slack.MsgOptionText(message.Text, false)); err != nil {
				return nil, err
			}
		}

		if message.Asset == nil || message.Asset.Filename == "" {
			continue
		}
		uploadRequest := slack.FileUploadParameters{
			Filename: message.Asset.Filename,
			Title:    message.Asset.Title,
			Filetype: message.Asset.Type,
			Content:  message.Asset.Content,
			Channels: []string{message.Channel},
		}
		if len(message.Asset.BinaryContent) > 0 {
			uploadRequest.Reader = bytes.NewReader(message.Asset.BinaryContent)
		}
		if _, err = client.UploadFile(uploadRequest); err != nil {
			return nil, err
		}
	}
	return response, nil
}

func (s *service) listen(context *endly.Context, request *ListenRequest) (*ListenResponse, error) {
	client, err := getClient(context, request.Credentials)
	if err != nil {
		return nil, err
	}
	wait := &sync.WaitGroup{}
	wait.Add(1)
	go s.listenInTheBackground(context, request, client, wait)
	wait.Wait()
	return &ListenResponse{}, nil
}

func getChannelId(channelName string, client *slack.Client) (string, error) {
	channelName = strings.Replace(channelName, "#", "", 1)
	channels, err := client.GetChannels(true)
	if err != nil {
		return "", err
	}
	available := []string{}
	for _, channel := range channels {
		if channel.Name == channelName {
			return channel.ID, nil
		}
		available = append(available, channel.Name)
	}
	return "", fmt.Errorf("failed to lookup channel %v, available: %v", channelName, available)
}

func (s *service) listenInTheBackground(context *endly.Context, request *ListenRequest, client *slack.Client, wait *sync.WaitGroup) {
	messages := s.getMessages()
	rtm := client.NewRTM()
	go rtm.ManageConnection()
	defer func() {
		if wait != nil {
			wait.Done()
		}
	}()
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.ConnectedEvent:
				if wait != nil {
					wait.Done()
					wait = nil
				}
			case *slack.MessageEvent:
				eventMessages, err := NewMessageFromEvent(ev, client)
				if err != nil {
					log.Printf("failed to create slack message: %v", err)
					continue
				}
				if len(eventMessages) == 0 {
					continue
				}
				for _, message := range eventMessages {
					messages.Push(message)
				}
			case *slack.InvalidAuthEvent:
				log.Print("unable to slack:listen invalid credentials")
				return

			}
		case <-time.After(time.Second):
			if context.IsClosed() {
				return
			}
		}
	}
}

func (s *service) pull(context *endly.Context, request *PullRequest) (*PullResponse, error) {
	response := &PullResponse{
		Messages: make([]*Message, 0),
	}
	var err error
	s.pullMessages(request, response)
	if request.Expect != nil {
		response.Assert, err = validator.Assert(context, request, request.Expect, response.Messages, "Slack.messages", "assert slack messages")
	}
	return response, err
}

func (s *service) pullMessages(request *PullRequest, response *PullResponse) {
	messages := s.getMessages()
	timeout := time.Millisecond * time.Duration(request.TimeoutMs)

	startTime := time.Now()
	for {
		message := messages.Shift()
		elapsed := time.Now().Sub(startTime)
		timeoutExceeded := elapsed >= timeout
		if message == nil {
			time.Sleep(200 * time.Millisecond)
			if timeoutExceeded {
				return
			}
			continue
		}
		response.Messages = append(response.Messages, message)
		if timeoutExceeded {
			return
		}
		if len(response.Messages) >= request.Count {
			return
		}
	}
}

func New() endly.Service {
	var result = &service{
		AbstractService: endly.NewAbstractService(ServiceID),
	}
	result.AbstractService.Service = result
	result.registerRoutes()
	return result
}
