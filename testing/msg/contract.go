package msg

import (
	"fmt"
	"github.com/viant/endly/testing/validator"
	"github.com/viant/toolbox/data"
	"github.com/viant/toolbox/url"
)

const defaultTimeoutMs = 10000

//CreateRequest represents a create resource request
type CreateRequest struct {
	Credentials string
	Resources   []*ResourceSetup
}

func (r *CreateRequest) Init() error {
	if len(r.Resources) == 0 {
		return nil
	}
	for _, resource := range r.Resources {
		if err := resource.Init(); err != nil {
			return err
		}
		if resource.Credentials == "" {
			resource.Credentials = r.Credentials
		}
	}
	return nil
}

func (r *CreateRequest) Validate() error {
	if len(r.Resources) == 0 {
		return fmt.Errorf("resources was empty")
	}
	for _, resource := range r.Resources {
		if err := resource.Validate(); err != nil {
			return err
		}

	}
	return nil
}

//CreateResponse represents a create resource response
type CreateResponse struct {
	Resources []*Resource
}

//DeleteRequest represents a delete resource request
type DeleteRequest struct {
	Credentials string
	Resources   []*Resource
}

func (r *DeleteRequest) Init() error {
	if len(r.Resources) == 0 {
		return nil
	}
	for _, resource := range r.Resources {
		_ = resource.Init()
		if resource.Credentials == "" {
			resource.Credentials = r.Credentials
		}
	}
	return nil
}

//DeleteResponse represents a delete resource response
type DeleteResponse struct{}

//PushRequest represents push request
type PushRequest struct {
	Credentials   string
	Dest          *Resource
	Messages      []*Message
	Source        *url.Resource `description:"source for message data  currently JSON or new line delimitered json are supported that conform []*Message type "`
	TimeoutMs     int
	UDF           string
	isInitialized bool
}

func (r *PushRequest) Init() error {
	if r.isInitialized {
		return nil
	}
	if r.Dest != nil {
		if err := r.Dest.Init(); err != nil {
			return err
		}
		if r.Dest.Credentials == "" {
			r.Dest.Credentials = r.Credentials
		}
	}
	if r.TimeoutMs == 0 {
		r.TimeoutMs = defaultTimeoutMs
	}
	return nil
}

func (r *PushRequest) Validate() error {
	if r.Dest == nil {
		return fmt.Errorf("dest was empty")
	}
	if len(r.Messages) == 0 && r.Source == nil {
		return fmt.Errorf("messages were empty")
	}
	return nil
}

//PushResponse represents a push response
type PushResponse struct {
	Results []Result
}

//PullRequest represents a pull request
type PullRequest struct {
	Credentials string
	Source      *Resource
	TimeoutMs   int
	Count       int
	Nack        bool `description:"flag indicates that the client will not or cannot process a Message passed to the Subscriber.Receive callback."`
	UDF         string
	Expect      interface{}
}

func (r *PullRequest) Init() error {
	if r.TimeoutMs == 0 {
		r.TimeoutMs = defaultTimeoutMs
	}
	if r.Source.Credentials == "" {
		r.Source.Credentials = r.Credentials
	}
	return r.Source.Init()
}

func (r *PullRequest) Validate() error {
	if r.Source == nil {
		return fmt.Errorf("source was empty")
	}
	return nil
}

//PullRequest represents a pull response
type PullResponse struct {
	Messages []*Message
	Assert   *validator.AssertResponse
}

type Message struct {
	ID          string
	Subject     string
	Attributes  map[string]interface{}
	Data        interface{}
	Transformed interface{} `description:"udf transformed data"`
}

func (m *Message) Expand(state data.Map) *Message {
	var result = &Message{
		Attributes: make(map[string]interface{}),
		Subject:    m.Subject,
	}
	if len(m.Attributes) > 0 {
		for k, v := range m.Attributes {
			result.Attributes[state.ExpandAsText(k)] = state.Expand(v)
		}
	}
	if m.Data != nil {
		result.Data = state.Expand(m.Data)
	}
	return result
}

type Result interface{}
