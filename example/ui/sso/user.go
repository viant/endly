package sso

import "time"

type User struct {
	Email          string     `json:"email" primaryKey:"true" column:"email"`
	Name           string     `json:"name" column:"name"`
	HashedPassword string     `json:"-" column:"hashedPassword"`
	DateOfBirth    *time.Time `json:"dateOfBirth" column:"dateOfBirth" `
}
