package smtp

import (
	"fmt"
	"github.com/emersion/go-smtp"
	"io"
)

type backend struct {
	messages *Messages
	users    []*User
}

func (b *backend) Login(username, password string) (smtp.User, error) {
	for _, candidate := range b.users {
		if candidate.Username == username {
			if candidate.Password == password {
				return &sMTPUser{messages: b.messages, username: username}, nil
			}
			break
		}
	}
	return nil, fmt.Errorf("invalid user or credentials: %v", username)
}

// Require clients to authenticate using SMTP AUTH before sending emails
func (bkd *backend) AnonymousLogin() (smtp.User, error) {
	return nil, smtp.ErrAuthRequired
}

func newBackend(messages *Messages, users []*User) smtp.Backend {
	return &backend{messages: messages, users: users}
}

type sMTPUser struct {
	username string
	messages *Messages
}

func (u *sMTPUser) Send(from string, to []string, r io.Reader) error {
	message, err := NewMessage(from, to, r)
	if err != nil {
		return err
	}
	u.messages.Push(u.username, message)
	return nil
}

func (u *sMTPUser) Logout() error {
	return nil
}
