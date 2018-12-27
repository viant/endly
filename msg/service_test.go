package msg

import (
	"github.com/stretchr/testify/assert"
	"github.com/viant/assertly"
	"github.com/viant/endly"
	"github.com/viant/toolbox"
	"log"
	"os"
	"path"
	"testing"
	"time"
)

func createResources(t *testing.T, resources ...*ResourceSetup) bool {
	var response = &CreateResponse{}
	err := endly.Run(nil, &CreateRequest{
		Resources: resources,
	}, response)
	if !assert.Nil(t, err) {
		log.Print(err)
		return false
	}
	return true
}

func deleteResource(t *testing.T, resourceSetup ...*ResourceSetup) bool {
	var response = &DeleteResponse{}

	var resources = []*Resource{}
	for _, setup := range resourceSetup {
		resources = append(resources, &setup.Resource)
	}
	err := endly.Run(nil, &DeleteRequest{
		Resources: resources,
	}, response)
	if !assert.Nil(t, err) {
		log.Print(err)
		return false
	}
	return true
}

func TestService_PushPull(t *testing.T) {
	var resources = []*ResourceSetup{
		NewResourceSetup("topic", "e2eTopic", "am", true, nil),
		NewResourceSetup("subscription", "e2eSubscription", "am", true, NewConfig("e2eTopic")),
	}

	if !createResources(t, resources...) {
		return
	}
	defer deleteResource(t, resources...)

	useCases := []struct {
		description string
		dest        *Resource
		source      *Resource
		messages    []*Message
		expected    interface{}
		hasError    bool
	}{
		{
			description: "google cloud push messages use case",
			dest:        NewResource("", "/projects/${pubsub.projectID}/topics/e2eTopic", "am"),
			source:      NewResource("", "/projects/${pubsub.projectID}/subscriptions/e2eSubscription", "am"),
			messages: []*Message{
				{
					Attributes: map[string]string{
						"attr1": "abc",
					},
					Data: "hello e2e topic",
				},
			},
		},
	}

	for _, useCase := range useCases {
		var credentialFile = path.Join(os.Getenv("HOME"), ".secret", useCase.dest.Credentials+".json")
		if !toolbox.FileExists(credentialFile) {
			//no secret file define skip the use case
			log.Printf("skipping test no credentials: " + credentialFile)
			continue
		}
		var pushResponse = &PushResponse{}
		err := endly.Run(nil, &PushRequest{
			Dest:     useCase.dest,
			Messages: useCase.messages,
		}, pushResponse)

		if useCase.hasError {
			assert.NotNil(t, err, useCase.description)
			continue
		}
		if !assert.Nil(t, err, useCase.description) {
			log.Print(err)
			continue
		}
		assert.EqualValues(t, len(useCase.messages), len(pushResponse.Results), useCase.description)

		time.Sleep(3 * time.Second)

		var pullResponse = &PullResponse{}
		err = endly.Run(nil, &PullRequest{
			Source: useCase.source,
			Count:  len(useCase.messages),
		}, pullResponse)

		if !assert.Nil(t, err, useCase.description) {
			log.Print(err)
			continue
		}
		assertly.AssertValues(t, useCase.messages, pullResponse.Messages)
	}

}
