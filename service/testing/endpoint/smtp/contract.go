package smtp

import (
	"context"
	"fmt"
	"github.com/viant/assertly"
	"github.com/viant/scy/cred/secret"
)

// ListenRequest represents a new listen request
type ListenRequest struct {
	ServerName   string
	Port         int
	EnableTLS    bool
	MaxBodySize  int
	Users        []*User
	CertLocation string
	Debug        bool
}

func (r *ListenRequest) Init() error {
	if r.ServerName == "" {
		r.ServerName = "localhost"
	}
	if r.Port == 0 {
		r.Port = 1587
	}
	if r.MaxBodySize == 0 {
		r.MaxBodySize = 1024 * 1024
	}
	if len(r.Users) > 0 {
		secretService := secret.New()
		for _, user := range r.Users {
			if user.Credentials != "" && user.Password == "" {
				if cred, err := secretService.GetCredentials(context.Background(), user.Credentials); err == nil {
					user.Password = cred.Password
					if user.Username == "" {
						user.Username = cred.Username
					}
				}

			}
		}
	}
	return nil
}

func (r *ListenRequest) Validate() error {
	if len(r.Users) == 0 {
		return fmt.Errorf("users were empty")
	}
	for i, user := range r.Users {
		if user.Password == "" {
			return fmt.Errorf("users[%d].Password was empty", i)
		}
	}
	return nil
}

// ListenResponse represents a new listen response
type ListenResponse struct{}

// UserMessage represents desired user message
type UserMessage struct {
	User    string
	TagID   string
	Message interface{}
}

// AssertRequest represents a log assert request
type AssertRequest struct {
	DescriptionTemplate string
	Expect              []*UserMessage `required:"true" description:"expected user messagesByUser"`
}

func (r *AssertRequest) Init() error {
	if r.DescriptionTemplate == "" {
		r.DescriptionTemplate = "Message Validation: $user"
	}
	return nil
}

// AssertResponse represents a smtp message assert response
type AssertResponse struct {
	Validations []*assertly.Validation
}

// Assertion returns description with validation slice
func (r *AssertResponse) Assertion() []*assertly.Validation {
	return r.Validations
}
