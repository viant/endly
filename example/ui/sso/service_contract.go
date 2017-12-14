package sso

import (
	"errors"
)

type SignUpRequest struct {
	*User
	DataOfBirth string
	Password string
	LandingPage string
}



type BaseResponse struct {
	Status string
	Error string
	ErrorSource string
	LandingPage string
}

func (r *SignUpRequest) Validate() (string, error) {
	if r.Name == "" {
		return "name", errors.New("name was empty")
	}
	if r.Email == "" {
		return "email", errors.New("email was empty")
	}
	if r.Password == "" {
		return "password", errors.New("password was empty")
	}
	if r.DataOfBirth == "" {
		return "dataOfBirth", errors.New("data of birth was empty")
	}
	return "", nil
}