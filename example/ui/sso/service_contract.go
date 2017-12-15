package sso

import (
	"errors"
)

type SignUpRequest struct {
	*User
	DataOfBirth string `json:"dateOfBirth"`
	Password    string `json:"password"`
	LandingPage string `json:"landingPage"`
}

type SignUpResponse struct {
	*BaseResponse
	*User
	LandingPage string `json:"landingPage"`
}


type SignInRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	RememberMe  bool   `json:"rememberMe"`
	LandingPage string `json:"landingPage"`
}

type SignInResponse struct {
	*BaseResponse
	*User
	LandingPage string `json:"landingPage"`
}

type BaseResponse struct {
	Status      string `json:"status"`
	Error       string `json:"error"`
	ErrorSource string `json:"errorSource"`
}

func (r *SignUpRequest) Validate() (string, error) {
	if r.User == nil || r.Name == "" {
		return "name", errors.New("name was empty")
	}
	if r.User == nil || r.Email == "" {
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
