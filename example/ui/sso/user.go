package sso

import "time"

// User represents a user
type User struct {
	Email          string     `json:"email" primaryKey:"true" column:"email"`
	Name           string     `json:"name" column:"name"`
	HashedPassword string     `json:"-" column:"hashedPassword"`
	DateOfBirth    *time.Time `json:"dateOfBirth" column:"dateOfBirth" `
}
