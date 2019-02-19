package slack

import (
	"bytes"
	"github.com/nlopes/slack"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/util/json"
)

//Asset represents a file asset
type Asset struct {
	Title         string
	Filename      string
	Type          string
	Content       string
	Data          interface{} //content data structure
	BinaryContent []byte
}

//Message represent a slack message
type Message struct {
	Channel  string
	Username string
	Text     string
	Asset    *Asset
}

//NewMessageFromEvent creates a new message form a message event
func NewMessageFromEvent(event *slack.MessageEvent, client *slack.Client) ([]*Message, error) {
	result := make([]*Message, 0)
	channel, err := client.GetChannelInfo(event.Channel)
	if err != nil {
		return nil, err
	}
	message := &Message{}
	message.Channel = channel.Name
	message.Text = event.Text
	message.Username = event.Username
	if event.Text != "" {
		result = append(result, message)
	}
	if len(event.Files) == 0 {
		return result, nil
	}

	for _, file := range event.Files {
		message := &Message{}
		message.Channel = channel.Name
		message.Asset = &Asset{
			Title:    file.Title,
			Filename: file.Name,
			Type:     file.Filetype,
			Content:  file.Preview,
		}
		if file.URLPrivateDownload != "" {
			buf := new(bytes.Buffer)
			if err := client.GetFile(file.URLPrivateDownload, buf); err != nil {
				return nil, err
			}
			message.Asset.Content = buf.String()
			if err != nil {
				return nil, err
			}
			if file.Filetype == "yaml" || file.Filetype == "yml" {
				_ = yaml.Unmarshal(buf.Bytes(), &message.Asset.Data)
			} else {
				_ = json.Unmarshal(buf.Bytes(), &message.Asset.Data)
			}

		}
		result = append(result, message)
	}
	return result, nil
}
