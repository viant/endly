package sso

import "time"

type User struct {
	Email          string     `json:"email",column:"email"`
	Name           string     `json:"name",column:"name"`
	HashedPassword string     `json:"-",column:"hashedPassword"`
	DataOfBirth    *time.Time `json:"dateOfBirth",column:"dateOfBirth"`
}
