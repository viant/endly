package sso

import "time"

type User struct {
	Email string
	Name string
	EncryptedPassword string
	DataOfBirth *time.Time
}
