package slack

import (
	"github.com/nlopes/slack"
	"github.com/viant/endly"
)

func getClient(context *endly.Context, credentials string) (*slack.Client, error) {
	credConfig, err := context.Secrets.GetCredentials(credentials)
	if err != nil {
		return nil, err
	}
	api := slack.New(credConfig.Password)
	return api, nil
}
