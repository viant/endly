package smtp

import (
	"fmt"
	"github.com/emersion/go-smtp"
	"io"
	"io/ioutil"
)

type backend struct {
	messages *Messages
	users    []*User
	state    *smtp.ConnectionState
}

func (b *backend) Login(state *smtp.ConnectionState, username, password string) (smtp.Session, error) {
	for _, candidate := range b.users {
		if candidate.Username == username {
			if candidate.Password == password {
				return &session{
					messages: b.messages,
					username: username,
					backend:b,
				}, nil
			}
			break
		}
	}
	return nil, fmt.Errorf("invalid user or credentials: %v", username)
}


// Called if the client attempts to send mail without logging in first.
// Return smtp.ErrAuthRequired if you don't want to support this.
func (b *backend) AnonymousLogin(state *smtp.ConnectionState) (smtp.Session, error) {
	return &session{
		messages: b.messages,
		username: "",
		backend:b,
	}, nil
}

func newBackend(messages *Messages, users []*User) smtp.Backend {
	return &backend{messages: messages, users: users}
}

type session struct {
	backend   *backend
	anonymous bool
	username  string
	messages  *Messages
	msg       *Message
}

func (s *session) Reset() {
	s.msg = &Message{
		To:make([]string, 0),
		Header:make(map[string]string),
	}
}

func (s *session) Logout() error {
	return nil
}

func (s *session) Mail(from string) error {
	s.Reset()
	s.msg.From = from
	return nil
}

func (s *session) Rcpt(to string) error {
	s.msg.To = append(s.msg.To, to)
	return nil
}

func (s *session) Data(r io.Reader) error {
	if b, err := ioutil.ReadAll(r); err != nil {
		return err
	} else {
		s.msg.Raw = string(b)
		s.msg.Decode()
		s.backend.messages.Push(s.username, s.msg)
	}
	return nil
}
